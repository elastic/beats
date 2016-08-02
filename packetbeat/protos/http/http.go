package http

import (
	"bytes"
	"expvar"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

var debugf = logp.MakeDebug("http")
var detailedf = logp.MakeDebug("httpdetailed")

type parserState uint8

const (
	stateStart parserState = iota
	stateFLine
	stateHeaders
	stateBody
	stateBodyChunkedStart
	stateBodyChunked
	stateBodyChunkedWaitFinalCRLF
)

var (
	unmatchedResponses = expvar.NewInt("http.unmatched_responses")
)

type stream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset  int
	parseState   parserState
	bodyReceived int

	message *message
}

type httpConnectionData struct {
	Streams   [2]*stream
	requests  messageList
	responses messageList
}

type messageList struct {
	head, tail *message
}

// HTTP application level protocol analyser plugin.
type HTTP struct {
	// config
	Ports               []int
	SendRequest         bool
	SendResponse        bool
	SplitCookie         bool
	HideKeywords        []string
	RedactAuthorization bool
	IncludeBodyFor      []string

	parserConfig parserConfig

	transactionTimeout time.Duration

	results publish.Transactions
}

var (
	isDebug    = false
	isDetailed = false
)

func init() {
	protos.Register("http", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &HTTP{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

// Init initializes the HTTP protocol analyser.
func (http *HTTP) init(results publish.Transactions, config *httpConfig) error {
	http.setFromConfig(config)

	isDebug = logp.IsDebug("http")
	isDetailed = logp.IsDebug("httpdetailed")
	http.results = results
	return nil
}

func (http *HTTP) setFromConfig(config *httpConfig) {
	http.Ports = config.Ports
	http.SendRequest = config.SendRequest
	http.SendResponse = config.SendResponse
	http.HideKeywords = config.Hide_keywords
	http.RedactAuthorization = config.Redact_authorization
	http.SplitCookie = config.Split_cookie
	http.parserConfig.RealIPHeader = strings.ToLower(config.Real_ip_header)
	http.transactionTimeout = config.TransactionTimeout
	http.IncludeBodyFor = config.Include_body_for

	if config.Send_all_headers {
		http.parserConfig.SendHeaders = true
		http.parserConfig.SendAllHeaders = true
	} else {
		if len(config.Send_headers) > 0 {
			http.parserConfig.SendHeaders = true

			http.parserConfig.HeadersWhitelist = map[string]bool{}
			for _, hdr := range config.Send_headers {
				http.parserConfig.HeadersWhitelist[strings.ToLower(hdr)] = true
			}
		}
	}
}

// GetPorts lists the port numbers the HTTP protocol analyser will handle.
func (http *HTTP) GetPorts() []int {
	return http.Ports
}

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Decides if we can ignore the gap or it's a parser error
// and we need to drop the stream.
func (http *HTTP) messageGap(s *stream, nbytes int) (ok bool, complete bool) {

	m := s.message
	switch s.parseState {
	case stateStart, stateHeaders:
		// we know we cannot recover from these
		return false, false
	case stateBody:
		if isDebug {
			debugf("gap in body: %d", nbytes)
		}

		if m.IsRequest {
			m.Notes = append(m.Notes, "Packet loss while capturing the request")
		} else {
			m.Notes = append(m.Notes, "Packet loss while capturing the response")
		}
		if !m.hasContentLength && (bytes.Equal(m.connection, constClose) ||
			(isVersion(m.version, 1, 0) && !bytes.Equal(m.connection, constKeepAlive))) {

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

func (st *stream) PrepareForNewMessage() {
	st.data = st.data[st.message.end:]
	st.parseState = stateStart
	st.parseOffset = 0
	st.bodyReceived = 0
	st.message = nil
}

// Called when the parser has identified the boundary
// of a message.
func (http *HTTP) messageComplete(
	conn *httpConnectionData,
	tcptuple *common.TcpTuple,
	dir uint8,
	st *stream,
) {
	st.message.Raw = st.data[st.message.start:st.message.end]

	http.handleHTTP(conn, st.message, tcptuple, dir)
}

// ConnectionTimeout returns the configured HTTP transaction timeout.
func (http *HTTP) ConnectionTimeout() time.Duration {
	return http.transactionTimeout
}

// Parse function is used to process TCP payloads.
func (http *HTTP) Parse(
	pkt *protos.Packet,
	tcptuple *common.TcpTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseHttp exception")

	conn := ensureHTTPConnection(private)
	conn = http.doParse(conn, pkt, tcptuple, dir)
	if conn == nil {
		return nil
	}
	return conn
}

func ensureHTTPConnection(private protos.ProtocolData) *httpConnectionData {
	conn := getHTTPConnection(private)
	if conn == nil {
		conn = &httpConnectionData{}
	}
	return conn
}

func getHTTPConnection(private protos.ProtocolData) *httpConnectionData {
	if private == nil {
		return nil
	}

	priv, ok := private.(*httpConnectionData)
	if !ok {
		logp.Warn("http connection data type error")
		return nil
	}
	if priv == nil {
		logp.Warn("Unexpected: http connection data not set")
		return nil
	}

	return priv
}

// Parse function is used to process TCP payloads.
func (http *HTTP) doParse(
	conn *httpConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TcpTuple,
	dir uint8,
) *httpConnectionData {

	if isDetailed {
		detailedf("Payload received: [%s]", pkt.Payload)
	}

	st := conn.Streams[dir]
	if st == nil {
		st = newStream(pkt, tcptuple)
		conn.Streams[dir] = st
	} else {
		// concatenate bytes
		st.data = append(st.data, pkt.Payload...)
		if len(st.data) > tcp.TCP_MAX_DATA_IN_STREAM {
			if isDebug {
				debugf("Stream data too large, dropping TCP stream")
			}
			conn.Streams[dir] = nil
			return conn
		}
	}

	for len(st.data) > 0 {
		if st.message == nil {
			st.message = &message{Ts: pkt.Ts}
		}

		parser := newParser(&http.parserConfig)
		ok, complete := parser.parse(st)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.Streams[dir] = nil
			return conn
		}

		if !complete {
			// wait for more data
			break
		}

		// all ok, ship it
		http.messageComplete(conn, tcptuple, dir, st)

		// and reset stream for next message
		st.PrepareForNewMessage()
	}

	return conn
}

func newStream(pkt *protos.Packet, tcptuple *common.TcpTuple) *stream {
	return &stream{
		tcptuple: tcptuple,
		data:     pkt.Payload,
		message:  &message{Ts: pkt.Ts},
	}
}

// ReceivedFin will be called when TCP transaction is terminating.
func (http *HTTP) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	conn := getHTTPConnection(private)
	if conn == nil {
		return private
	}

	stream := conn.Streams[dir]
	if stream == nil {
		return conn
	}

	// send whatever data we got so far as complete. This
	// is needed for the HTTP/1.0 without Content-Length situation.
	if stream.message != nil && len(stream.data[stream.message.start:]) > 0 {
		stream.message.Raw = stream.data[stream.message.start:]
		http.handleHTTP(conn, stream.message, tcptuple, dir)

		// and reset message. Probably not needed, just to be sure.
		stream.PrepareForNewMessage()
	}

	return conn
}

// GapInStream is called when a gap of nbytes bytes is found in the stream (due
// to packet loss).
func (http *HTTP) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(http) exception")

	conn := getHTTPConnection(private)
	if conn == nil {
		return private, false
	}

	stream := conn.Streams[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	ok, complete := http.messageGap(stream, nbytes)
	if isDetailed {
		detailedf("messageGap returned ok=%v complete=%v", ok, complete)
	}
	if !ok {
		// on errors, drop stream
		conn.Streams[dir] = nil
		return conn, true
	}

	if complete {
		// Current message is complete, we need to publish from here
		http.messageComplete(conn, tcptuple, dir, stream)
	}

	// don't drop the stream, we can ignore the gap
	return private, false
}

func (http *HTTP) handleHTTP(
	conn *httpConnectionData,
	m *message,
	tcptuple *common.TcpTuple,
	dir uint8,
) {

	m.TCPTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())
	http.hideHeaders(m)

	if m.IsRequest {
		if isDebug {
			debugf("Received request with tuple: %s", m.TCPTuple)
		}
		conn.requests.append(m)
	} else {
		if isDebug {
			debugf("Received response with tuple: %s", m.TCPTuple)
		}
		conn.responses.append(m)
		http.correlate(conn)
	}
}

func (http *HTTP) correlate(conn *httpConnectionData) {
	// drop responses with missing requests
	if conn.requests.empty() {
		for !conn.responses.empty() {
			debugf("Response from unknown transaction. Ingoring.")
			unmatchedResponses.Add(1)
			conn.responses.pop()
		}
		return
	}

	// merge requests with responses into transactions
	for !conn.responses.empty() && !conn.requests.empty() {
		requ := conn.requests.pop()
		resp := conn.responses.pop()
		trans := http.newTransaction(requ, resp)

		if isDebug {
			debugf("HTTP transaction completed")
		}
		http.publishTransaction(trans)
	}
}

func (http *HTTP) newTransaction(requ, resp *message) common.MapStr {
	status := common.OK_STATUS
	if resp.StatusCode >= 400 {
		status = common.ERROR_STATUS
	}

	// resp_time in milliseconds
	responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

	path, params, err := http.extractParameters(requ, requ.Raw)
	if err != nil {
		logp.Warn("Fail to parse HTTP parameters: %v", err)
	}

	src := common.Endpoint{
		Ip:   requ.TCPTuple.Src_ip.String(),
		Port: requ.TCPTuple.Src_port,
		Proc: string(requ.CmdlineTuple.Src),
	}
	dst := common.Endpoint{
		Ip:   requ.TCPTuple.Dst_ip.String(),
		Port: requ.TCPTuple.Dst_port,
		Proc: string(requ.CmdlineTuple.Dst),
	}
	if requ.Direction == tcp.TcpDirectionReverse {
		src, dst = dst, src
	}

	http_details := common.MapStr{
		"request": common.MapStr{
			"params":  params,
			"headers": http.collectHeaders(requ),
		},
		"response": common.MapStr{
			"code":    resp.StatusCode,
			"phrase":  resp.StatusPhrase,
			"headers": http.collectHeaders(resp),
		},
	}

	http.setBody(http_details["request"].(common.MapStr), requ)
	http.setBody(http_details["response"].(common.MapStr), resp)

	event := common.MapStr{
		"@timestamp":   common.Time(requ.Ts),
		"type":         "http",
		"status":       status,
		"responsetime": responseTime,
		"method":       requ.Method,
		"path":         path,
		"query":        fmt.Sprintf("%s %s", requ.Method, path),
		"http":         http_details,
		"bytes_out":    resp.Size,
		"bytes_in":     requ.Size,
		"src":          &src,
		"dst":          &dst,
	}

	if http.SendRequest {
		event["request"] = string(http.cutMessageBody(requ))
	}
	if http.SendResponse {
		event["response"] = string(http.cutMessageBody(resp))
	}

	if len(requ.Notes)+len(resp.Notes) > 0 {
		event["notes"] = append(requ.Notes, resp.Notes...)
	}
	if len(requ.RealIP) > 0 {
		event["real_ip"] = requ.RealIP
	}

	return event
}

func (http *HTTP) publishTransaction(event common.MapStr) {
	if http.results == nil {
		return
	}
	http.results.PublishTransaction(event)
}

func (http *HTTP) collectHeaders(m *message) interface{} {

	hdrs := map[string]interface{}{}

	hdrs["content-length"] = m.ContentLength
	if len(m.ContentType) > 0 {
		hdrs["content-type"] = m.ContentType
	}

	if http.parserConfig.SendHeaders {

		cookie := "cookie"
		if !m.IsRequest {
			cookie = "set-cookie"
		}

		for name, value := range m.Headers {
			if strings.ToLower(name) == "content-type" {
				continue
			}
			if strings.ToLower(name) == "content-length" {
				continue
			}
			if http.SplitCookie {
				if name == cookie {
					hdrs[name] = splitCookiesHeader(string(value))
				}
			} else {
				hdrs[name] = value
			}
		}
	}
	fmt.Println("Headers: ", hdrs)
	return hdrs
}

func (http *HTTP) setBody(result common.MapStr, m *message) {
	body := string(http.extractBody(m))
	if len(body) > 0 {
		result["body"] = body
	}
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

func parseCookieValue(raw string) string {
	// Strip the quotes, if present.
	if len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	return raw
}

func (http *HTTP) extractBody(m *message) []byte {
	body := []byte{}

	if len(m.ContentType) == 0 || http.shouldIncludeInBody(m.ContentType) {
		if len(m.chunkedBody) > 0 {
			body = append(body, m.chunkedBody...)
		} else {
			if isDebug {
				debugf("Body to include: [%s]", m.Raw[m.bodyOffset:])
			}
			body = append(body, m.Raw[m.bodyOffset:]...)
		}
	}

	return body
}

func (http *HTTP) cutMessageBody(m *message) []byte {
	cutMsg := []byte{}

	// add headers always
	cutMsg = m.Raw[:m.bodyOffset]

	// add body
	return append(cutMsg, http.extractBody(m)...)

}

func (http *HTTP) shouldIncludeInBody(contenttype []byte) bool {
	includedBodies := http.IncludeBodyFor
	for _, include := range includedBodies {
		if bytes.Contains(contenttype, []byte(include)) {
			if isDebug {
				debugf("Should Include Body = true Content-Type %s include_body %s",
					contenttype, include)
			}
			return true
		}
		if isDebug {
			debugf("Should Include Body = false Content-Type %s include_body %s",
				contenttype, include)
		}
	}
	return false
}

func (http *HTTP) hideHeaders(m *message) {
	if !m.IsRequest || !http.RedactAuthorization {
		return
	}

	msg := m.Raw

	// byte64 != encryption, so obscure it in headers in case of Basic Authentication

	redactHeaders := []string{"authorization", "proxy-authorization"}
	authText := []byte("uthorization:") // [aA] case insensitive, also catches Proxy-Authorization:

	authHeaderStartX := m.headerOffset
	authHeaderEndX := m.bodyOffset

	for authHeaderStartX < m.bodyOffset {
		if isDebug {
			debugf("looking for authorization from %d to %d",
				authHeaderStartX, authHeaderEndX)
		}

		startOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], authText)
		if startOfHeader >= 0 {
			authHeaderStartX = authHeaderStartX + startOfHeader

			endOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], []byte("\r\n"))
			if endOfHeader >= 0 {
				authHeaderEndX = authHeaderStartX + endOfHeader

				if authHeaderEndX > m.bodyOffset {
					authHeaderEndX = m.bodyOffset
				}

				if isDebug {
					debugf("Redact authorization from %d to %d", authHeaderStartX, authHeaderEndX)
				}

				for i := authHeaderStartX + len(authText); i < authHeaderEndX; i++ {
					msg[i] = byte('*')
				}
			}
		}
		authHeaderStartX = authHeaderEndX + len("\r\n")
		authHeaderEndX = m.bodyOffset
	}

	for _, header := range redactHeaders {
		if len(m.Headers[header]) > 0 {
			m.Headers[header] = []byte("*")
		}
	}

	m.Raw = msg
}

func (http *HTTP) hideSecrets(values url.Values) url.Values {
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
// Returns the Request URI path and the (adjusted) parameters.
func (http *HTTP) extractParameters(m *message, msg []byte) (path string, params string, err error) {
	var values url.Values

	u, err := url.Parse(string(m.RequestURI))
	if err != nil {
		return
	}
	values = u.Query()
	path = u.Path

	paramsMap := http.hideSecrets(values)

	if m.ContentLength > 0 && bytes.Contains(m.ContentType, []byte("urlencoded")) {

		values, err = url.ParseQuery(string(msg[m.bodyOffset:]))
		if err != nil {
			return
		}

		for key, value := range http.hideSecrets(values) {
			paramsMap[key] = value
		}
	}

	params = paramsMap.Encode()
	if isDetailed {
		detailedf("Form parameters: %s", params)
	}
	return
}

func (http *HTTP) isSecretParameter(key string) bool {
	for _, keyword := range http.HideKeywords {
		if strings.ToLower(key) == keyword {
			return true
		}
	}
	return false
}

func (ml *messageList) append(msg *message) {
	if ml.tail == nil {
		ml.head = msg
	} else {
		ml.tail.next = msg
	}
	msg.next = nil
	ml.tail = msg
}

func (ml *messageList) empty() bool {
	return ml.head == nil
}

func (ml *messageList) pop() *message {
	if ml.head == nil {
		return nil
	}

	msg := ml.head
	ml.head = ml.head.next
	if ml.head == nil {
		ml.tail = nil
	}
	return msg
}

func (ml *messageList) last() *message {
	return ml.tail
}
