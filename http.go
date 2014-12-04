package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"labix.org/v2/mgo/bson"
)

const (
	START = iota
	FLINE
	HEADERS
	BODY
	BODY_CHUNKED_START
	BODY_CHUNKED
)

// Http Message
type HttpMessage struct {
	Ts               time.Time
	hasContentLength bool
	bodyOffset       int
	version_major    uint8
	version_minor    uint8
	connection       string
	chunked_length   int
	chunked_body     []byte

	IsRequest    bool
	TcpTuple     TcpTuple
	CmdlineTuple *CmdlineTuple
	Direction    uint8
	//Request Info
	FirstLine    string
	RequestUri   string
	Method       string
	StatusCode   uint16
	StatusPhrase string
	Real_ip      string
	// Http Headers
	ContentLength    int
	TransferEncoding string
	Headers          map[string]string
	Body             string
	//Raw Data
	Raw []byte
	//Timing
	start int
	end   int
}

type HttpStream struct {
	tcpStream *TcpStream

	data []byte

	parseOffset  int
	parseState   int
	bodyReceived int

	message *HttpMessage
}

type HttpTransaction struct {
	Type         string
	tuple        TcpTuple
	Src          Endpoint
	Dst          Endpoint
	Real_ip      string
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	cmdline      *CmdlineTuple

	Http bson.M

	Request_raw  string
	Response_raw string

	timer *time.Timer
}

type Http struct {
	// config
	Send_request      bool
	Send_response     bool
	Send_headers      bool
	Send_all_headers  bool
	Headers_whitelist map[string]bool
	Split_cookie      bool
	Real_ip_header    string

	transactionsMap map[HashableTcpTuple]*HttpTransaction

	Publisher *PublisherType
}

type tomlHttp struct {
	Send_all_headers bool
	Send_headers     []string
	Split_cookie     bool
	Real_ip_header   string
}

var HttpMod Http

func (http *Http) InitDefaults() {
	http.Send_request = true
	http.Send_response = true
}

func (http *Http) setFromConfig() (err error) {
	if _ConfigMeta.IsDefined("protocols", "http", "send_request") {
		http.Send_request = _Config.Protocols["http"].Send_request
	}
	if _ConfigMeta.IsDefined("protocols", "http", "send_response") {
		http.Send_response = _Config.Protocols["http"].Send_response
	}

	if _Config.Http.Send_all_headers {
		http.Send_headers = true
		http.Send_all_headers = true
	} else {
		if len(_Config.Http.Send_headers) > 0 {
			http.Send_headers = true

			http.Headers_whitelist = map[string]bool{}
			for _, hdr := range _Config.Http.Send_headers {
				http.Headers_whitelist[strings.ToLower(hdr)] = true
			}
		}
	}

	http.Split_cookie = _Config.Http.Split_cookie

	http.Real_ip_header = strings.ToLower(_Config.Http.Real_ip_header)

	return nil
}

func (http *Http) Init(test_mode bool) error {

	http.InitDefaults()

	if !test_mode {
		err := http.setFromConfig()
		if err != nil {
			return err
		}
	}

	http.transactionsMap = make(map[HashableTcpTuple]*HttpTransaction, TransactionsHashSize)

	if !test_mode {
		http.Publisher = &Publisher
	}

	return nil
}

func parseVersion(s []byte) (uint8, uint8, error) {
	if len(s) < 3 {
		return 0, 0, MsgError("Invalid version")
	}

	major, _ := strconv.Atoi(string(s[0]))
	minor, _ := strconv.Atoi(string(s[2]))

	return uint8(major), uint8(minor), nil
}

func parseResponseStatus(s []byte) (uint16, string, error) {

	DEBUG("http", "parseResponseStatus: %s", s)

	p := bytes.Index(s, []byte(" "))
	if p == -1 {
		return 0, "", MsgError("Not beeing able to identify status code")
	}

	status_code, _ := strconv.Atoi(string(s[0:p]))

	p = bytes.LastIndex(s, []byte(" "))
	if p == -1 {
		return uint16(status_code), "", MsgError("Not beeing able to identify status code")
	}
	status_phrase := s[p+1:]
	return uint16(status_code), string(status_phrase), nil
}

func (http *Http) parseHeader(m *HttpMessage, data []byte) (bool, bool, int) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	i := bytes.Index(data, []byte(":"))
	if i == -1 {
		// Expected \":\" in headers. Assuming incomplete"
		return true, false, 0
	}

	DEBUG("httpdetailed", "Data: %s", data)
	DEBUG("httpdetailed", "Header: %s", data[:i])

	// skip folding line
	for p := i + 1; p < len(data); {
		q := bytes.Index(data[p:], []byte("\r\n"))
		if q == -1 {
			// Assuming incomplete
			return true, false, 0
		}
		p += q
		DEBUG("httpdetailed", "HV: %s\n", data[i+1:p])
		if len(data) > p && (data[p+1] == ' ' || data[p+1] == '\t') {
			p = p + 2
		} else {
			headerName := strings.ToLower(string(data[:i]))
			headerVal := string(bytes.Trim(data[i+1:p], " \t"))
			DEBUG("http", "Header: '%s' Value: '%s'\n", headerName, headerVal)

			// Headers we need for parsing. Make sure we always
			// capture their value
			if headerName == "content-length" {
				m.ContentLength, _ = strconv.Atoi(headerVal)
			} else if headerName == "transfer-encoding" {
				m.TransferEncoding = headerVal
			}
			if len(http.Real_ip_header) > 0 && headerName == http.Real_ip_header {
				m.Real_ip = headerVal
			}

			if http.Send_headers {
				if !http.Send_all_headers {
					_, exists := http.Headers_whitelist[headerName]
					if !exists {
						return true, true, p + 2
					}
				}
				if val, ok := m.Headers[headerName]; ok {
					m.Headers[headerName] = val + ", " + headerVal
				} else {
					m.Headers[headerName] = headerVal
				}
			}

			return true, true, p + 2
		}
	}

	return true, false, len(data)
}

func (http *Http) messageParser(s *HttpStream) (bool, bool) {

	var cont, ok, complete bool
	m := s.message

	DEBUG("http", "Stream state=%d", s.parseState)

	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case START:
			m.start = s.parseOffset
			i := bytes.Index(s.data[s.parseOffset:], []byte("\r\n"))
			if i == -1 {
				return true, false
			}

			// Very basic tests on the first line. Just to check that
			// we have what looks as an HTTP message
			var version []byte
			var err error
			fline := s.data[s.parseOffset:i]
			if len(fline) < 8 {
				DEBUG("http", "First line too small")
				return false, false
			}
			if bytes.Equal(fline[0:5], []byte("HTTP/")) {
				//RESPONSE
				m.IsRequest = false
				version = fline[5:8]
				m.StatusCode, m.StatusPhrase, err = parseResponseStatus(fline[9:])
				if err != nil {
					WARN("Failed to understand HTTP response status: %s", fline[9:])
					return false, false
				}
				DEBUG("http", "HTTP status_code=%d, status_phrase=%s", m.StatusCode, m.StatusPhrase)

			} else {
				// REQUEST
				slices := bytes.Fields(fline)
				if len(slices) != 3 {
					DEBUG("http", "Couldn't understand HTTP request: %s", fline)
					return false, false
				}

				m.Method = string(slices[0])
				m.RequestUri = string(slices[1])

				if bytes.Equal(slices[2][:5], []byte("HTTP/")) {
					m.IsRequest = true
					version = slices[2][5:]
					m.FirstLine = string(fline)
				} else {
					DEBUG("http", "Couldn't understand HTTP version: %s", fline)
					return false, false
				}
				DEBUG("http", "HTTP Method=%s, RequestUri=%s", m.Method, m.RequestUri)
			}

			m.version_major, m.version_minor, err = parseVersion(version)
			if err != nil {
				DEBUG("http", "Failed to understand HTTP version: %s", version)
				m.version_major = 1
				m.version_minor = 0
			}
			DEBUG("http", "HTTP version %d.%d", m.version_major, m.version_minor)

			// ok so far
			s.parseOffset = i + 2
			s.parseState = HEADERS

		case HEADERS:

			if len(s.data)-s.parseOffset >= 2 &&
				bytes.Equal(s.data[s.parseOffset:s.parseOffset+2], []byte("\r\n")) {
				// EOH
				s.parseOffset += 2
				m.bodyOffset = s.parseOffset
				if m.ContentLength == 0 {
					if m.version_major == 1 && m.version_minor == 0 &&
						!m.hasContentLength {
						if m.IsRequest {
							// No Content-Length in a HTTP/1.0 request means
							// there is no body
							m.end = s.parseOffset
							return true, true
						} else {
							// Read until FIN
						}
					} else if m.connection == "close" {
						// Connection: close -> read until FIN
					} else if !m.hasContentLength && m.TransferEncoding == "chunked" {
						// support for HTTP/1.1 Chunked transfer
						s.parseState = BODY_CHUNKED_START
						continue
					} else {
						m.end = s.parseOffset
						return true, true
					}
				}
				s.parseState = BODY
			} else {
				ok, hfcomplete, offset := http.parseHeader(m, s.data[s.parseOffset:])

				if !ok {
					return false, false
				}
				if !hfcomplete {
					return true, false
				}
				s.parseOffset += offset
			}

		case BODY:
			DEBUG("http", "eat body: %d", s.parseOffset)
			if !m.hasContentLength && m.connection == "close" {
				// HTTP/1.0 no content length. Add until the end of the connection
				DEBUG("http", "close connection, %d", len(s.data)-s.parseOffset)
				s.bodyReceived += (len(s.data) - s.parseOffset)
				m.ContentLength += (len(s.data) - s.parseOffset)
				s.parseOffset = len(s.data)
				return true, false
			} else if len(s.data[s.parseOffset:]) >= m.ContentLength-s.bodyReceived {
				s.parseOffset += (m.ContentLength - s.bodyReceived)
				m.end = s.parseOffset
				return true, true
			} else {
				s.bodyReceived += (len(s.data) - s.parseOffset)
				s.parseOffset = len(s.data)
				return true, false
			}

		case BODY_CHUNKED_START:
			cont, ok, complete = state_body_chunked_start(s, m)
			if !cont {
				return ok, complete
			}

		case BODY_CHUNKED:
			cont, ok, complete = state_body_chunked(s, m)
			if !cont {
				return ok, complete
			}
		}

	}

	return true, false
}

func state_body_chunked_start(s *HttpStream, m *HttpMessage) (cont bool, ok bool, complete bool) {
	// read hexa length
	i := bytes.Index(s.data[s.parseOffset:], []byte("\r\n"))
	if i == -1 {
		return false, true, false
	}
	line := string(s.data[s.parseOffset : s.parseOffset+i])
	_, err := fmt.Sscanf(line, "%x", &m.chunked_length)
	if err != nil {
		WARN("Failed to understand chunked body start line")
		return false, false, false
	}

	s.parseOffset += i + 2 //+ \r\n
	if m.chunked_length == 0 {
		s.parseOffset += 2 // final \r\n
		m.end = s.parseOffset
		return false, true, true
	}
	s.bodyReceived = 0
	s.parseState = BODY_CHUNKED

	return true, false, false
}

func state_body_chunked(s *HttpStream, m *HttpMessage) (cont bool, ok bool, complete bool) {
	if len(s.data[s.parseOffset:]) >= m.chunked_length-s.bodyReceived+2 /*\r\n*/ {
		// Received more data than expected
		m.chunked_body = append(m.chunked_body, s.data[s.parseOffset:s.parseOffset+m.chunked_length-s.bodyReceived]...)
		s.parseOffset += (m.chunked_length - s.bodyReceived + 2 /*\r\n*/)
		m.ContentLength += m.chunked_length
		s.parseState = BODY_CHUNKED_START
		return true, false, false
	} else {
		if len(s.data[s.parseOffset:]) >= m.chunked_length-s.bodyReceived {
			// we need need to wait for the +2, else we can crash on next call
			return false, true, false
		}
		// Received less data than expected
		m.chunked_body = append(m.chunked_body, s.data[s.parseOffset:]...)
		s.bodyReceived += (len(s.data) - s.parseOffset)
		s.parseOffset = len(s.data)
		return false, true, false
	}
	return true, false, false
}

func (stream *HttpStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.message.end:]
	stream.parseState = START
	stream.parseOffset = 0
	stream.bodyReceived = 0
	stream.message = nil
}

func (http *Http) Parse(pkt *Packet, tcp *TcpStream, dir uint8) {
	defer RECOVER("ParseHttp exception")

	DEBUG("http", "Payload received: [%s]", pkt.payload)

	if tcp.httpData[dir] == nil {
		tcp.httpData[dir] = &HttpStream{
			tcpStream: tcp,
			data:      pkt.payload,
			message:   &HttpMessage{Ts: pkt.ts},
		}

	} else {
		// concatenate bytes
		tcp.httpData[dir].data = append(tcp.httpData[dir].data, pkt.payload...)
		if len(tcp.httpData[dir].data) > TCP_MAX_DATA_IN_STREAM {
			DEBUG("http", "Stream data too large, dropping TCP stream")
			tcp.httpData[dir] = nil
			return
		}
	}
	stream := tcp.httpData[dir]
	if stream.message == nil {
		stream.message = &HttpMessage{Ts: pkt.ts}
	}
	ok, complete := http.messageParser(stream)

	if !ok {
		// drop this tcp stream. Will retry parsing with the next
		// segment in it
		tcp.httpData[dir] = nil
		return
	}

	if complete {
		// all ok, ship it
		msg := stream.data[stream.message.start:stream.message.end]
		censorPasswords(stream.message, msg)

		http.handleHttp(stream.message, tcp, dir, msg)

		// and reset message
		stream.PrepareForNewMessage()
	}
}

func (http *Http) ReceivedFin(tcp *TcpStream, dir uint8) {
	if tcp.httpData[dir] == nil {
		return
	}

	stream := tcp.httpData[dir]

	// send whatever data we got so far as complete. This
	// is needed for the HTTP/1.0 without Content-Length situation.
	if stream.message != nil &&
		len(stream.data[stream.message.start:]) > 0 {

		DEBUG("httpdetailed", "Publish something on connection FIN")

		msg := stream.data[stream.message.start:]
		censorPasswords(stream.message, msg)

		http.handleHttp(stream.message, tcp, dir, msg)

		// and reset message. Probably not needed, just to be sure.
		stream.PrepareForNewMessage()
	}
}

func (http *Http) handleHttp(m *HttpMessage, tcp *TcpStream,
	dir uint8, raw_msg []byte) {

	m.TcpTuple = TcpTupleFromIpPort(tcp.tuple, tcp.id)
	m.Direction = dir
	m.CmdlineTuple = procWatcher.FindProcessesTuple(tcp.tuple)
	m.Raw = raw_msg

	if m.IsRequest {
		http.receivedHttpRequest(m)
	} else {
		http.receivedHttpResponse(m)
	}
}

func (http *Http) receivedHttpRequest(msg *HttpMessage) {

	trans := http.transactionsMap[msg.TcpTuple.raw]
	if trans != nil {
		if len(trans.Http) != 0 {
			WARN("Two requests without a response. Dropping old request")
		}
	} else {
		trans = &HttpTransaction{Type: "http", tuple: msg.TcpTuple}
		http.transactionsMap[msg.TcpTuple.raw] = trans
	}

	DEBUG("http", "Received request with tuple: %s", msg.TcpTuple)

	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000)
	trans.JsTs = msg.Ts
	trans.Src = Endpoint{
		Ip:   msg.TcpTuple.Src_ip.String(),
		Port: msg.TcpTuple.Src_port,
		Proc: string(msg.CmdlineTuple.Src),
	}
	trans.Dst = Endpoint{
		Ip:   msg.TcpTuple.Dst_ip.String(),
		Port: msg.TcpTuple.Dst_port,
		Proc: string(msg.CmdlineTuple.Dst),
	}
	if msg.Direction == TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}

	// save Raw message
	if http.Send_request {
		trans.Request_raw = string(cutMessageBody(msg))
	}

	request := bson.M{
		"method":   msg.Method,
		"uri":      msg.RequestUri,
		"uri.raw":  msg.RequestUri,
		"line":     msg.FirstLine,
		"line.raw": msg.FirstLine,
	}

	if http.Send_headers {
		if !http.Split_cookie {
			request["headers"] = msg.Headers
		} else {
			hdrs := bson.M{}
			for hdr_name, hdr_val := range msg.Headers {
				if hdr_name == "cookie" {
					hdrs[hdr_name] = splitCookiesHeader(hdr_val)
				} else {
					hdrs[hdr_name] = hdr_val
				}
			}

			request["headers"] = hdrs
		}
	}

	trans.Http = bson.M{
		"request": request,
	}

	trans.Real_ip = msg.Real_ip

	if trans.timer != nil {
		trans.timer.Stop()
	}
	trans.timer = time.AfterFunc(TransactionTimeout, func() { http.expireTransaction(trans) })

}

func (http *Http) expireTransaction(trans *HttpTransaction) {
	// remove from map
	delete(http.transactionsMap, trans.tuple.raw)
}

func (http *Http) receivedHttpResponse(msg *HttpMessage) {

	// we need to search the request first.
	tuple := msg.TcpTuple

	DEBUG("http", "Received response with tuple: %s", tuple)

	trans := http.transactionsMap[tuple.raw]
	if trans == nil {
		WARN("Response from unknown transaction. Ignoring: %v", tuple)
		return
	}

	if len(trans.Http) == 0 {
		WARN("Response without a known request. Ignoring.")
		return
	}

	response := bson.M{
		"phrase": msg.StatusPhrase,
		"code":   msg.StatusCode,
	}

	if http.Send_headers {
		if !http.Split_cookie {
			response["headers"] = msg.Headers
		} else {
			hdrs := bson.M{}
			for hdr_name, hdr_val := range msg.Headers {
				if hdr_name == "set-cookie" {
					hdrs[hdr_name] = splitCookiesHeader(hdr_val)
				} else {
					hdrs[hdr_name] = hdr_val
				}
			}

			response["headers"] = hdrs
		}
	}

	trans.Http = bson_concat(trans.Http, bson.M{
		"content_length": msg.ContentLength,
		"response":       response,
	})

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if http.Send_response {
		trans.Response_raw = string(cutMessageBody(msg))
	}

	err := http.PublishTransaction(trans)

	if err != nil {
		WARN("Publish failure: %s", err)
	}

	DEBUG("http", "HTTP transaction completed: %s -> %s\n", trans.Http["request"],
		trans.Http["response"])

	// remove from map
	delete(http.transactionsMap, trans.tuple.raw)
	if trans.timer != nil {
		trans.timer.Stop()
	}
}

func (http *Http) PublishTransaction(t *HttpTransaction) error {

	if http.Publisher == nil {
		return nil
	}

	event := Event{}

	event.Type = "http"
	response := t.Http["response"].(bson.M)
	code := response["code"].(uint16)
	if code < 400 {
		event.Status = OK_STATUS
	} else {
		event.Status = ERROR_STATUS
	}
	event.ResponseTime = t.ResponseTime
	if http.Send_request {
		event.RequestRaw = t.Request_raw
	}
	if http.Send_response {
		event.ResponseRaw = t.Response_raw
	}
	event.Http = t.Http
	event.Real_ip = t.Real_ip

	return http.Publisher.PublishEvent(t.ts, &t.Src, &t.Dst, &event)

}

func splitCookiesHeader(headerVal string) map[string]string {
	cookies := map[string]string{}

	cstring := strings.Split(headerVal, ";")
	for _, cval := range cstring {
		cookie := strings.Split(cval, "=")
		cookies[strings.ToLower(strings.Trim(cookie[0], " "))] = cookie[1]
	}

	return cookies
}

func cutMessageBody(m *HttpMessage) []byte {
	raw_msg_cut := []byte{}

	// add headers always
	raw_msg_cut = m.Raw[:m.bodyOffset]

	// add body
	contentType, ok := m.Headers["content-type"]
	if ok && (len(contentType) == 0 || shouldIncludeInBody(contentType)) {
		if len(m.chunked_body) > 0 {
			raw_msg_cut = append(raw_msg_cut, m.chunked_body...)
		} else {
			raw_msg_cut = append(raw_msg_cut, m.Raw[m.bodyOffset:]...)
		}
	}

	return raw_msg_cut
}

func shouldIncludeInBody(contenttype string) bool {
	include_body := _Config.ContentTypes.Include_body
	for _, include := range include_body {
		if strings.Contains(contenttype, include) {
			DEBUG("http", "Should Include Body = true Content-Type "+contenttype+" include_body "+include)
			return true
		}
		DEBUG("http", "Should Include Body = false Content-Type"+contenttype+" include_body "+include)
	}
	return false
}

func censorPasswords(m *HttpMessage, msg []byte) {

	keywords := _Config.Passwords.Hide_keywords

	if m.IsRequest && m.ContentLength > 0 &&
		strings.Contains(m.Headers["content-type"], "urlencoded") {
		for _, keyword := range keywords {
			index := bytes.Index(msg[m.bodyOffset:], []byte(keyword))
			if index > 0 {
				start_index := m.bodyOffset + index + len(keyword)
				end_index := bytes.IndexAny(msg[m.bodyOffset+index+len(keyword):], "& \r\n")
				if end_index > 0 {
					end_index += m.bodyOffset + index
					if end_index > m.end {
						end_index = m.end
					}
				} else {
					end_index = m.end
				}

				if end_index-start_index < 120 {
					for i := start_index; i < end_index; i++ {
						msg[i] = byte('*')
					}
				}
			}
		}
	}
}
