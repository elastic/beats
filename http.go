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

type HttpMessage struct {
	Ts                time.Time
	hasContentLength  bool
	bodyOffset        int
	version_major     uint8
	version_minor     uint8
	connection        string
	transfer_encoding string
	chunked_length    int
	chunked_body      []byte

	IsRequest     bool
	Method        string
	StatusCode    uint16
	Host          string
	RequestUri    string
	FirstLine     string
	TcpTuple      TcpTuple
	CmdlineTuple  *CmdlineTuple
	Direction     uint8
	ContentLength int
	ContentType   string
	ReasonPhrase  string
	XForwardedFor string

	Raw []byte

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

var transactionsMap = make(map[HashableTcpTuple]*HttpTransaction, TransactionsHashSize)

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
	reason_phrase := s[p+1:]
	return uint16(status_code), string(reason_phrase), nil
}

func httpParseHeader(m *HttpMessage, data []byte) (bool, bool, int) {

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

			if bytes.Equal(bytes.ToLower(data[:i]), []byte("host")) {
				m.Host = string(bytes.Trim(data[i+1:p], " \t"))
			} else if bytes.Equal(bytes.ToLower(data[:i]), []byte("content-length")) {
				m.ContentLength, _ = strconv.Atoi(string(bytes.Trim(data[i+1:p], " \t")))
				m.hasContentLength = true
			} else if bytes.Equal(bytes.ToLower(data[:i]), []byte("content-type")) {
				m.ContentType = string(bytes.Trim(data[i+1:p], " \t"))
			} else if bytes.Equal(bytes.ToLower(data[:i]), []byte("connection")) {
				m.connection = string(bytes.Trim(data[i+1:p], " \t"))
			} else if bytes.Equal(bytes.ToLower(data[:i]), []byte("transfer-encoding")) {
				m.transfer_encoding = string(bytes.Trim(data[i+1:p], " \t"))
			} else if bytes.Equal(bytes.ToLower(data[:i]), []byte("x-forwarded-for")) {
				m.XForwardedFor = string(bytes.Trim(data[i+1:p], " \t"))
			}

			return true, true, p + 2
		}
	}

	return true, false, len(data)
}

func httpMessageParser(s *HttpStream) (bool, bool) {

	var cont, ok, complete bool
	m := s.message

	DEBUG("http", "Stream state=%d", s.parseState)

	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case START:
			m.start = s.parseOffset
			i := bytes.Index(s.data, []byte("\r\n"))
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
				m.StatusCode, m.ReasonPhrase, err = parseResponseStatus(fline[9:])
				if err != nil {
					WARN("Failed to understand HTTP response status: %s", fline[9:])
					return false, false
				}
				DEBUG("http", "HTTP status_code=%d, reason_phrase=%s", m.StatusCode, m.ReasonPhrase)

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
					} else if !m.hasContentLength && m.transfer_encoding == "chunked" {
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
				ok, hfcomplete, offset := httpParseHeader(m, s.data[s.parseOffset:])

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

func ParseHttp(pkt *Packet, tcp *TcpStream, dir uint8) {
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

	ok, complete := httpMessageParser(stream)

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

		handleHttp(stream.message, tcp, dir, msg)

		// and reset message
		stream.PrepareForNewMessage()
	}
}

func HttpReceivedFin(tcp *TcpStream, dir uint8) {
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

		handleHttp(stream.message, tcp, dir, msg)

		// and reset message. Probably not needed, just to be sure.
		stream.PrepareForNewMessage()
	}
}

func handleHttp(m *HttpMessage, tcp *TcpStream,
	dir uint8, raw_msg []byte) {

	m.TcpTuple = TcpTupleFromIpPort(tcp.tuple, tcp.id)
	m.Direction = dir
	m.CmdlineTuple = procWatcher.FindProcessesTuple(tcp.tuple)
	m.Raw = raw_msg

	if m.IsRequest {
		receivedHttpRequest(m)
	} else {
		receivedHttpResponse(m)
	}
}

func receivedHttpRequest(msg *HttpMessage) {

	trans := transactionsMap[msg.TcpTuple.raw]
	if trans != nil {
		if len(trans.Http) != 0 {
			WARN("Two requests without a response. Dropping old request")
		}
	} else {
		trans = &HttpTransaction{Type: "http", tuple: msg.TcpTuple}
		transactionsMap[msg.TcpTuple.raw] = trans
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
	trans.Request_raw = string(cutMessageBody(msg))

	trans.Http = bson.M{
		"host": msg.Host,
		"request": bson.M{
			"method":          msg.Method,
			"uri":             msg.RequestUri,
			"uri.raw":         msg.RequestUri,
			"line":            msg.FirstLine,
			"line.raw":        msg.FirstLine,
			"x-forwarded-for": msg.XForwardedFor,
		},
	}

	if trans.timer != nil {
		trans.timer.Stop()
	}
	trans.timer = time.AfterFunc(TransactionTimeout, func() { trans.Expire() })

}

func (trans *HttpTransaction) Expire() {
	// remove from map
	delete(transactionsMap, trans.tuple.raw)
}

func receivedHttpResponse(msg *HttpMessage) {

	// we need to search the request first.
	tuple := msg.TcpTuple

	DEBUG("http", "Received response with tuple: %s", tuple)

	trans := transactionsMap[tuple.raw]
	if trans == nil {
		WARN("Response from unknown transaction. Ignoring: %v", tuple)
		return
	}

	if len(trans.Http) == 0 {
		WARN("Response without a known request. Ignoring.")
		return
	}

	trans.Http = bson_concat(trans.Http, bson.M{
		"content_length": msg.ContentLength,
		"content_type":   msg.ContentType,
		"response": bson.M{
			"code":   msg.StatusCode,
			"phrase": msg.ReasonPhrase,
		},
	})

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	trans.Response_raw = string(cutMessageBody(msg))

	err := Publisher.PublishHttpTransaction(trans)

	if err != nil {
		WARN("Publish failure: %s", err)
	}

	DEBUG("http", "HTTP transaction completed: %s -> %s\n", trans.Http["request"],
		trans.Http["response"])

	// remove from map
	delete(transactionsMap, trans.tuple.raw)
	if trans.timer != nil {
		trans.timer.Stop()
	}
}

func cutMessageBody(m *HttpMessage) []byte {
	raw_msg_cut := []byte{}

	// add headers always
	raw_msg_cut = m.Raw[:m.bodyOffset]

	// add body
	if len(m.ContentType) == 0 || shouldIncludeInBody(m.ContentType) {
		if len(m.chunked_body) > 0 {
			raw_msg_cut = append(raw_msg_cut, m.chunked_body...)
		} else {
			raw_msg_cut = append(raw_msg_cut, m.Raw[m.bodyOffset:]...)
		}
	}

	return raw_msg_cut
}

func shouldIncludeInBody(contenttype string) bool {
	return strings.Contains(contenttype, "form-urlencoded") ||
		strings.Contains(contenttype, "json")
}

func censorPasswords(m *HttpMessage, msg []byte) {

	keywords := _Config.Passwords.Hide_keywords

	if m.IsRequest && m.ContentLength > 0 &&
		strings.Contains(m.ContentType, "urlencoded") {
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
