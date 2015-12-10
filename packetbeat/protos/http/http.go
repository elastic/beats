package http

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

const (
	START = iota
	FLINE
	HEADERS
	BODY
	BODY_CHUNKED_START
	BODY_CHUNKED
	BODY_CHUNKED_WAIT_FINAL_CRLF
)

// Http Message
type HttpMessage struct {
	Ts               time.Time
	hasContentLength bool
	headerOffset     int
	bodyOffset       int
	version_major    uint8
	version_minor    uint8
	connection       string
	chunked_length   int
	chunked_body     []byte

	IsRequest    bool
	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
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
	ContentType      string
	TransferEncoding string
	Headers          map[string]string
	Body             string
	Size             uint64
	//Raw Data
	Raw []byte

	Notes []string

	//Timing
	start int
	end   int
}

type HttpStream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset  int
	parseState   int
	bodyReceived int

	message *HttpMessage
}

type HttpTransaction struct {
	Type         string
	tuple        common.TcpTuple
	Src          common.Endpoint
	Dst          common.Endpoint
	Real_ip      string
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	cmdline      *common.CmdlineTuple
	Method       string
	RequestUri   string
	Params       string
	Path         string
	BytesOut     uint64
	BytesIn      uint64
	Notes        []string

	Http common.MapStr

	Request_raw  string
	Response_raw string
}

type Http struct {
	// config
	Ports                []int
	Send_request         bool
	Send_response        bool
	Send_headers         bool
	Send_all_headers     bool
	Headers_whitelist    map[string]bool
	Split_cookie         bool
	Real_ip_header       string
	Hide_keywords        []string
	Redact_authorization bool

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publisher.Client
}

func (http *Http) getTransaction(k common.HashableTcpTuple) *HttpTransaction {
	v := http.transactions.Get(k)
	if v != nil {
		return v.(*HttpTransaction)
	}
	return nil
}

func (http *Http) InitDefaults() {
	http.Send_request = false
	http.Send_response = false
	http.Redact_authorization = false
	http.transactionTimeout = protos.DefaultTransactionExpiration
}

func (http *Http) SetFromConfig(config config.Http) (err error) {

	http.Ports = config.Ports

	if config.SendRequest != nil {
		http.Send_request = *config.SendRequest
	}
	if config.SendResponse != nil {
		http.Send_response = *config.SendResponse
	}
	http.Hide_keywords = config.Hide_keywords
	if config.Redact_authorization != nil {
		http.Redact_authorization = *config.Redact_authorization
	}

	if config.Send_all_headers != nil {
		http.Send_headers = true
		http.Send_all_headers = true
	} else {
		if len(config.Send_headers) > 0 {
			http.Send_headers = true

			http.Headers_whitelist = map[string]bool{}
			for _, hdr := range config.Send_headers {
				http.Headers_whitelist[strings.ToLower(hdr)] = true
			}
		}
	}

	if config.Split_cookie != nil {
		http.Split_cookie = *config.Split_cookie
	}

	if config.Real_ip_header != nil {
		http.Real_ip_header = strings.ToLower(*config.Real_ip_header)
	}

	if config.TransactionTimeout != nil && *config.TransactionTimeout > 0 {
		http.transactionTimeout = time.Duration(*config.TransactionTimeout) * time.Second
	}

	return nil
}

func (http *Http) GetPorts() []int {
	return http.Ports
}

func (http *Http) Init(test_mode bool, results publisher.Client) error {
	http.InitDefaults()

	if !test_mode {
		err := http.SetFromConfig(config.ConfigSingleton.Protocols.Http)
		if err != nil {
			return err
		}
	}

	http.transactions = common.NewCache(
		http.transactionTimeout,
		protos.DefaultTransactionHashSize)
	http.transactions.StartJanitor(http.transactionTimeout)
	http.results = results

	return nil
}

func parseVersion(s []byte) (uint8, uint8, error) {
	if len(s) < 3 {
		return 0, 0, errors.New("Invalid version")
	}

	major, _ := strconv.Atoi(string(s[0]))
	minor, _ := strconv.Atoi(string(s[2]))

	return uint8(major), uint8(minor), nil
}

func parseResponseStatus(s []byte) (uint16, string, error) {

	logp.Debug("http", "parseResponseStatus: %s", s)

	p := bytes.Index(s, []byte(" "))
	if p == -1 {
		return 0, "", errors.New("Not beeing able to identify status code")
	}

	status_code, _ := strconv.Atoi(string(s[0:p]))

	p = bytes.LastIndex(s, []byte(" "))
	if p == -1 {
		return uint16(status_code), "", errors.New("Not beeing able to identify status code")
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

	logp.Debug("httpdetailed", "Data: %s", data)
	logp.Debug("httpdetailed", "Header: %s", data[:i])

	// skip folding line
	for p := i + 1; p < len(data); {
		q := bytes.Index(data[p:], []byte("\r\n"))
		if q == -1 {
			// Assuming incomplete
			return true, false, 0
		}
		p += q
		logp.Debug("httpdetailed", "HV: %s\n", data[i+1:p])
		if len(data) > p && (data[p+1] == ' ' || data[p+1] == '\t') {
			p = p + 2
		} else {
			headerName := strings.ToLower(string(data[:i]))
			headerVal := string(bytes.Trim(data[i+1:p], " \t"))
			logp.Debug("http", "Header: '%s' Value: '%s'\n", headerName, headerVal)

			// Headers we need for parsing. Make sure we always
			// capture their value
			if headerName == "content-length" {
				m.ContentLength, _ = strconv.Atoi(headerVal)
				m.hasContentLength = true
			} else if headerName == "content-type" {
				m.ContentType = headerVal
			} else if headerName == "transfer-encoding" {
				m.TransferEncoding = headerVal
			} else if headerName == "connection" {
				m.connection = headerVal
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
				logp.Debug("http", "First line too small")
				return false, false
			}
			if bytes.Equal(fline[0:5], []byte("HTTP/")) {
				//RESPONSE
				m.IsRequest = false
				version = fline[5:8]
				m.StatusCode, m.StatusPhrase, err = parseResponseStatus(fline[9:])
				if err != nil {
					logp.Warn("Failed to understand HTTP response status: %s", fline[9:])
					return false, false
				}
				logp.Debug("http", "HTTP status_code=%d, status_phrase=%s", m.StatusCode, m.StatusPhrase)

			} else {
				// REQUEST
				slices := bytes.Fields(fline)
				if len(slices) != 3 {
					logp.Debug("http", "Couldn't understand HTTP request: %s", fline)
					return false, false
				}

				m.Method = string(slices[0])
				m.RequestUri = string(slices[1])

				if bytes.Equal(slices[2][:5], []byte("HTTP/")) {
					m.IsRequest = true
					version = slices[2][5:]
					m.FirstLine = string(fline)
				} else {
					logp.Debug("http", "Couldn't understand HTTP version: %s", fline)
					return false, false
				}
				logp.Debug("http", "HTTP Method=%s, RequestUri=%s", m.Method, m.RequestUri)
			}

			m.version_major, m.version_minor, err = parseVersion(version)
			if err != nil {
				logp.Debug("http", "Failed to understand HTTP version: %s", version)
				m.version_major = 1
				m.version_minor = 0
			}
			logp.Debug("http", "HTTP version %d.%d", m.version_major, m.version_minor)

			// ok so far
			s.parseOffset = i + 2
			m.headerOffset = s.parseOffset
			s.parseState = HEADERS

		case HEADERS:

			if len(s.data)-s.parseOffset >= 2 &&
				bytes.Equal(s.data[s.parseOffset:s.parseOffset+2], []byte("\r\n")) {
				// EOH
				s.parseOffset += 2
				m.bodyOffset = s.parseOffset
				if !m.IsRequest && ((100 <= m.StatusCode && m.StatusCode < 200) || m.StatusCode == 204 || m.StatusCode == 304) {
					//response with a 1xx, 204 , or 304 status  code is always terminated
					// by the first empty line after the  header fields
					logp.Debug("http", "Terminate response, status code %d", m.StatusCode)
					m.end = s.parseOffset
					m.Size = uint64(m.end - m.start)
					return true, true
				}
				if m.TransferEncoding == "chunked" {
					// support for HTTP/1.1 Chunked transfer
					// Transfer-Encoding overrides the Content-Length
					logp.Debug("http", "Read chunked body")
					s.parseState = BODY_CHUNKED_START
					continue
				}
				if m.ContentLength == 0 && (m.IsRequest || m.hasContentLength) {
					logp.Debug("http", "Empty content length, ignore body")
					// Ignore body for request that contains a message body but not a Content-Length
					m.end = s.parseOffset
					m.Size = uint64(m.end - m.start)
					return true, true
				}
				logp.Debug("http", "Read body")
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
			logp.Debug("http", "eat body: %d", s.parseOffset)
			if !m.hasContentLength && (m.connection == "close" ||
				(m.version_major == 1 && m.version_minor == 0 &&
					m.connection != "keep-alive")) {

				// HTTP/1.0 no content length. Add until the end of the connection
				logp.Debug("http", "close connection, %d", len(s.data)-s.parseOffset)
				s.bodyReceived += (len(s.data) - s.parseOffset)
				m.ContentLength += (len(s.data) - s.parseOffset)
				s.parseOffset = len(s.data)
				return true, false
			} else if len(s.data[s.parseOffset:]) >= m.ContentLength-s.bodyReceived {
				s.parseOffset += (m.ContentLength - s.bodyReceived)
				m.end = s.parseOffset
				m.Size = uint64(m.end - m.start)
				return true, true
			} else {
				s.bodyReceived += (len(s.data) - s.parseOffset)
				s.parseOffset = len(s.data)
				logp.Debug("http", "bodyReceived: %d", s.bodyReceived)
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

		case BODY_CHUNKED_WAIT_FINAL_CRLF:
			return state_body_chunked_wait_final_crlf(s, m)
		}

	}

	return true, false
}

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Decides if we can ignore the gap or it's a parser error
// and we need to drop the stream.
func (http *Http) messageGap(s *HttpStream, nbytes int) (ok bool, complete bool) {

	m := s.message
	switch s.parseState {
	case START, HEADERS:
		// we know we cannot recover from these
		return false, false
	case BODY:
		logp.Debug("http", "gap in body: %d", nbytes)
		if m.IsRequest {
			m.Notes = append(m.Notes, "Packet loss while capturing the request")
		} else {
			m.Notes = append(m.Notes, "Packet loss while capturing the response")
		}
		if !m.hasContentLength && (m.connection == "close" ||
			(m.version_major == 1 && m.version_minor == 0 &&
				m.connection != "keep-alive")) {

			s.bodyReceived += nbytes
			m.ContentLength += nbytes
			return true, false
		} else if len(s.data[s.parseOffset:])+nbytes >= m.ContentLength-s.bodyReceived {
			// we're done, but the last portion of the data is gone
			m.end = s.parseOffset
			return true, true
		} else {
			s.bodyReceived += nbytes
			return true, false
		}
	}
	// assume we cannot recover
	return false, false
}

func state_body_chunked_wait_final_crlf(s *HttpStream, m *HttpMessage) (ok bool, complete bool) {
	if len(s.data[s.parseOffset:]) < 2 {
		return true, false
	} else {
		if s.data[s.parseOffset] != '\r' || s.data[s.parseOffset+1] != '\n' {
			logp.Warn("Expected CRLF sequence at end of message")
			return false, false
		}
		s.parseOffset += 2 // skip final CRLF
		m.end = s.parseOffset
		m.Size = uint64(m.end - m.start)
		return true, true
	}
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
		logp.Warn("Failed to understand chunked body start line")
		return false, false, false
	}

	s.parseOffset += i + 2 //+ \r\n
	if m.chunked_length == 0 {
		if len(s.data[s.parseOffset:]) < 2 {
			s.parseState = BODY_CHUNKED_WAIT_FINAL_CRLF
			return false, true, false
		}
		if s.data[s.parseOffset] != '\r' || s.data[s.parseOffset+1] != '\n' {
			logp.Warn("Expected CRLF sequence at end of message")
			return false, false, false
		}
		s.parseOffset += 2 // skip final CRLF

		m.end = s.parseOffset
		m.Size = uint64(m.end - m.start)
		return false, true, true
	}
	s.bodyReceived = 0
	s.parseState = BODY_CHUNKED

	return true, true, false
}

func state_body_chunked(s *HttpStream, m *HttpMessage) (cont bool, ok bool, complete bool) {

	if len(s.data[s.parseOffset:]) >= m.chunked_length-s.bodyReceived+2 /*\r\n*/ {
		// Received more data than expected
		m.chunked_body = append(m.chunked_body, s.data[s.parseOffset:s.parseOffset+m.chunked_length-s.bodyReceived]...)
		s.parseOffset += (m.chunked_length - s.bodyReceived + 2 /*\r\n*/)
		m.ContentLength += m.chunked_length
		s.parseState = BODY_CHUNKED_START
		return true, true, false
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
}

func (stream *HttpStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.message.end:]
	stream.parseState = START
	stream.parseOffset = 0
	stream.bodyReceived = 0
	stream.message = nil
}

type httpPrivateData struct {
	Data [2]*HttpStream
}

// Called when the parser has identified the boundary
// of a message.
func (http *Http) messageComplete(tcptuple *common.TcpTuple, dir uint8, stream *HttpStream) {
	msg := stream.data[stream.message.start:stream.message.end]
	http.hideHeaders(stream.message, msg)

	http.handleHttp(stream.message, tcptuple, dir, msg)

	// and reset message
	stream.PrepareForNewMessage()
}

func (http *Http) ConnectionTimeout() time.Duration {
	return http.transactionTimeout
}

func (http *Http) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParseHttp exception")

	logp.Debug("httpdetailed", "Payload received: [%s]", pkt.Payload)

	priv := httpPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(httpPrivateData)
		if !ok {
			priv = httpPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &HttpStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &HttpMessage{Ts: pkt.Ts},
		}

	} else {
		// concatenate bytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("http", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}
	stream := priv.Data[dir]
	if stream.message == nil {
		stream.message = &HttpMessage{Ts: pkt.Ts}
	}
	ok, complete := http.messageParser(stream)

	if !ok {
		// drop this tcp stream. Will retry parsing with the next
		// segment in it
		priv.Data[dir] = nil
		return priv
	}

	if complete {
		// all ok, ship it
		http.messageComplete(tcptuple, dir, stream)
	}

	return priv
}

func (http *Http) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	if private == nil {
		return private
	}
	httpData, ok := private.(httpPrivateData)
	if !ok {
		return private
	}
	if httpData.Data[dir] == nil {
		return httpData
	}

	stream := httpData.Data[dir]

	// send whatever data we got so far as complete. This
	// is needed for the HTTP/1.0 without Content-Length situation.
	if stream.message != nil &&
		len(stream.data[stream.message.start:]) > 0 {

		logp.Debug("httpdetailed", "Publish something on connection FIN")

		msg := stream.data[stream.message.start:]
		http.hideHeaders(stream.message, msg)

		http.handleHttp(stream.message, tcptuple, dir, msg)

		// and reset message. Probably not needed, just to be sure.
		stream.PrepareForNewMessage()
	}

	return httpData
}

// Called when a gap of nbytes bytes is found in the stream (due to
// packet loss).
func (http *Http) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(http) exception")

	if private == nil {
		return private, false
	}
	httpData, ok := private.(httpPrivateData)
	if !ok {
		return private, false
	}
	stream := httpData.Data[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	ok, complete := http.messageGap(stream, nbytes)
	logp.Debug("httpdetailed", "messageGap returned ok=%v complete=%v", ok, complete)
	if !ok {
		// on errors, drop stream
		httpData.Data[dir] = nil
		return httpData, true
	}

	if complete {
		// Current message is complete, we need to publish from here
		http.messageComplete(tcptuple, dir, stream)
	}

	// don't drop the stream, we can ignore the gap
	return private, false
}

func (http *Http) handleHttp(m *HttpMessage, tcptuple *common.TcpTuple,
	dir uint8, raw_msg []byte) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())
	m.Raw = raw_msg

	if m.IsRequest {
		http.receivedHttpRequest(m)
	} else {
		http.receivedHttpResponse(m)
	}
}

func (http *Http) receivedHttpRequest(msg *HttpMessage) {

	trans := http.getTransaction(msg.TcpTuple.Hashable())
	if trans != nil {
		if len(trans.Http) != 0 {
			logp.Warn("Two requests without a response. Dropping old request")
		}
	} else {
		trans = &HttpTransaction{Type: "http", tuple: msg.TcpTuple}
		http.transactions.Put(msg.TcpTuple.Hashable(), trans)
	}

	logp.Debug("http", "Received request with tuple: %s", msg.TcpTuple)

	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000)
	trans.JsTs = msg.Ts
	trans.Src = common.Endpoint{
		Ip:   msg.TcpTuple.Src_ip.String(),
		Port: msg.TcpTuple.Src_port,
		Proc: string(msg.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   msg.TcpTuple.Dst_ip.String(),
		Port: msg.TcpTuple.Dst_port,
		Proc: string(msg.CmdlineTuple.Dst),
	}
	if msg.Direction == tcp.TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}

	// save Raw message
	if http.Send_request {
		trans.Request_raw = string(http.cutMessageBody(msg))
	}

	trans.Method = msg.Method
	trans.RequestUri = msg.RequestUri
	trans.BytesIn = msg.Size
	trans.Notes = msg.Notes

	trans.Http = common.MapStr{}

	if http.Send_headers {
		if !http.Split_cookie {
			trans.Http["request_headers"] = msg.Headers
		} else {
			hdrs := common.MapStr{}
			for hdr_name, hdr_val := range msg.Headers {
				if hdr_name == "cookie" {
					hdrs[hdr_name] = splitCookiesHeader(hdr_val)
				} else {
					hdrs[hdr_name] = hdr_val
				}
			}

			trans.Http["request_headers"] = hdrs
		}
	}

	trans.Real_ip = msg.Real_ip

	var err error
	trans.Path, trans.Params, err = http.extractParameters(msg, msg.Raw)
	if err != nil {
		logp.Warn("http", "Fail to parse HTTP parameters: %v", err)
	}
}

func (http *Http) receivedHttpResponse(msg *HttpMessage) {

	// we need to search the request first.
	tuple := msg.TcpTuple

	logp.Debug("http", "Received response with tuple: %s", tuple)

	trans := http.getTransaction(tuple.Hashable())
	if trans == nil {
		logp.Warn("Response from unknown transaction. Ignoring: %v", tuple)
		return
	}

	if trans.Http == nil {
		logp.Warn("Response without a known request. Ignoring.")
		return
	}

	response := common.MapStr{
		"phrase":         msg.StatusPhrase,
		"code":           msg.StatusCode,
		"content_length": msg.ContentLength,
	}

	if http.Send_headers {
		if !http.Split_cookie {
			response["response_headers"] = msg.Headers
		} else {
			hdrs := common.MapStr{}
			for hdr_name, hdr_val := range msg.Headers {
				if hdr_name == "set-cookie" {
					hdrs[hdr_name] = splitCookiesHeader(hdr_val)
				} else {
					hdrs[hdr_name] = hdr_val
				}
			}

			response["response_headers"] = hdrs
		}
	}

	trans.BytesOut = msg.Size
	trans.Http.Update(response)
	trans.Notes = append(trans.Notes, msg.Notes...)

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if http.Send_response {
		trans.Response_raw = string(http.cutMessageBody(msg))
	}

	http.publishTransaction(trans)
	http.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("http", "HTTP transaction completed: %s\n", trans.Http)
}

func (http *Http) publishTransaction(t *HttpTransaction) {

	if http.results == nil {
		return
	}

	event := common.MapStr{}

	event["type"] = "http"
	code := t.Http["code"].(uint16)
	if code < 400 {
		event["status"] = common.OK_STATUS
	} else {
		event["status"] = common.ERROR_STATUS
	}
	event["responsetime"] = t.ResponseTime
	if http.Send_request {
		event["request"] = t.Request_raw
	}
	if http.Send_response {
		event["response"] = t.Response_raw
	}
	event["http"] = t.Http
	if len(t.Real_ip) > 0 {
		event["real_ip"] = t.Real_ip
	}
	event["method"] = t.Method
	event["path"] = t.Path
	event["query"] = fmt.Sprintf("%s %s", t.Method, t.Path)
	event["params"] = t.Params

	event["bytes_out"] = t.BytesOut
	event["bytes_in"] = t.BytesIn
	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	if len(t.Notes) > 0 {
		event["notes"] = t.Notes
	}

	http.results.PublishEvent(event)
}

func parseCookieValue(raw string) string {
	// Strip the quotes, if present.
	if len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	return raw
}

func splitCookiesHeader(headerVal string) map[string]string {
	cookies := map[string]string{}

	cstring := strings.Split(headerVal, ";")
	for _, cval := range cstring {
		cookie := strings.SplitN(cval, "=", 2)
		if len(cookie) == 2 {
			cookies[strings.ToLower(strings.TrimSpace(cookie[0]))] =
				parseCookieValue(strings.TrimSpace(cookie[1]))
		}
	}

	return cookies
}

func (http *Http) cutMessageBody(m *HttpMessage) []byte {
	raw_msg_cut := []byte{}

	// add headers always
	raw_msg_cut = m.Raw[:m.bodyOffset]

	// add body
	if len(m.ContentType) == 0 || http.shouldIncludeInBody(m.ContentType) {
		if len(m.chunked_body) > 0 {
			raw_msg_cut = append(raw_msg_cut, m.chunked_body...)
		} else {
			logp.Debug("http", "Body to include: [%s]", m.Raw[m.bodyOffset:])
			raw_msg_cut = append(raw_msg_cut, m.Raw[m.bodyOffset:]...)
		}
	}

	return raw_msg_cut
}

func (http *Http) shouldIncludeInBody(contenttype string) bool {
	include_body := config.ConfigSingleton.Protocols.Http.Include_body_for
	for _, include := range include_body {
		if strings.Contains(contenttype, include) {
			logp.Debug("http", "Should Include Body = true Content-Type "+contenttype+" include_body "+include)
			return true
		}
		logp.Debug("http", "Should Include Body = false Content-Type"+contenttype+" include_body "+include)
	}
	return false
}

func (http *Http) hideHeaders(m *HttpMessage, msg []byte) {

	if m.IsRequest {
		// byte64 != encryption, so obscure it in headers in case of Basic Authentication
		if http.Redact_authorization {

			redactHeaders := []string{"authorization", "proxy-authorization"}
			auth_text := []byte("uthorization:") // [aA] case insensitive, also catches Proxy-Authorization:

			authHeaderStartX := m.headerOffset
			authHeaderEndX := m.bodyOffset

			for authHeaderStartX < m.bodyOffset {
				logp.Debug("http", "looking for authorization from %d to %d", authHeaderStartX, authHeaderEndX)

				startOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], auth_text)
				if startOfHeader >= 0 {
					authHeaderStartX = authHeaderStartX + startOfHeader

					endOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], []byte("\r\n"))
					if endOfHeader >= 0 {
						authHeaderEndX = authHeaderStartX + endOfHeader

						if authHeaderEndX > m.bodyOffset {
							authHeaderEndX = m.bodyOffset
						}

						logp.Debug("http", "Redact authorization from %d to %d", authHeaderStartX, authHeaderEndX)

						for i := authHeaderStartX + len(auth_text); i < authHeaderEndX; i++ {
							msg[i] = byte('*')
						}
					}
				}
				authHeaderStartX = authHeaderEndX + len("\r\n")
				authHeaderEndX = m.bodyOffset
			}
			for _, header := range redactHeaders {
				if m.Headers[header] != "" {
					m.Headers[header] = "*"
				}
			}
		}
	}
}

func (http *Http) hideSecrets(values url.Values) url.Values {

	params := url.Values{}
	for key, array := range values {
		for _, value := range array {
			if http.isSecretParameter(key) {
				params.Add(key, "xxxxx")
			} else {
				params.Add(key, value)
			}
		}
	}
	return params
}

// extractParameters parses the URL and the form parameters and replaces the secrets
// with the string xxxxx. The parameters containing secrets are defined in http.Hide_secrets.
// Returns the Request URI path and the (ajdusted) parameters.
func (http *Http) extractParameters(m *HttpMessage, msg []byte) (path string, params string, err error) {
	var values url.Values

	u, err := url.Parse(m.RequestUri)
	if err != nil {
		return
	}
	values = u.Query()
	path = u.Path

	paramsMap := http.hideSecrets(values)

	if m.ContentLength > 0 && strings.Contains(m.ContentType, "urlencoded") {
		values, err = url.ParseQuery(string(msg[m.bodyOffset:]))
		if err != nil {
			return
		}

		for key, value := range http.hideSecrets(values) {
			paramsMap[key] = value
		}
	}
	params = paramsMap.Encode()

	logp.Debug("httpdetailed", "Parameters: %s", params)

	return
}

func (http *Http) isSecretParameter(key string) bool {

	for _, keyword := range http.Hide_keywords {
		if strings.ToLower(key) == keyword {
			return true
		}
	}
	return false
}
