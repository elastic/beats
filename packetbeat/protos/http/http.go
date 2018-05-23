package http

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
)

var debugf = logp.MakeDebug("http")
var detailedf = logp.MakeDebug("httpdetailed")

type parserState uint8

const (
	stateStart parserState = iota
	stateHeaders
	stateBody
	stateBodyChunkedStart
	stateBodyChunked
	stateBodyChunkedWaitFinalCRLF
)

var (
	unmatchedResponses = monitoring.NewInt(nil, "http.unmatched_responses")
	unmatchedRequests  = monitoring.NewInt(nil, "http.unmatched_requests")
)

type stream struct {
	tcptuple *common.TCPTuple

	data []byte

	parseOffset  int
	parseState   parserState
	bodyReceived int

	message *message
}

type httpConnectionData struct {
	streams   [2]*stream
	requests  messageList
	responses messageList
}

type messageList struct {
	head, tail *message
}

// HTTP application level protocol analyser plugin.
type httpPlugin struct {
	// config
	ports               []int
	sendRequest         bool
	sendResponse        bool
	splitCookie         bool
	hideKeywords        []string
	redactAuthorization bool
	maxMessageSize      int

	parserConfig parserConfig

	transactionTimeout time.Duration

	results protos.Reporter
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
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &httpPlugin{}
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
func (http *httpPlugin) init(results protos.Reporter, config *httpConfig) error {
	http.setFromConfig(config)

	isDebug = logp.IsDebug("http")
	isDetailed = logp.IsDebug("httpdetailed")
	http.results = results
	return nil
}

func (http *httpPlugin) setFromConfig(config *httpConfig) {
	http.ports = config.Ports
	http.sendRequest = config.SendRequest
	http.sendResponse = config.SendResponse
	http.hideKeywords = config.HideKeywords
	http.redactAuthorization = config.RedactAuthorization
	http.splitCookie = config.SplitCookie
	http.parserConfig.realIPHeader = strings.ToLower(config.RealIPHeader)
	http.transactionTimeout = config.TransactionTimeout
	for _, list := range [][]string{config.IncludeBodyFor, config.IncludeRequestBodyFor} {
		http.parserConfig.includeRequestBodyFor = append(http.parserConfig.includeRequestBodyFor, list...)
	}
	for _, list := range [][]string{config.IncludeBodyFor, config.IncludeResponseBodyFor} {
		http.parserConfig.includeResponseBodyFor = append(http.parserConfig.includeResponseBodyFor, list...)
	}
	http.maxMessageSize = config.MaxMessageSize

	if config.SendAllHeaders {
		http.parserConfig.sendHeaders = true
		http.parserConfig.sendAllHeaders = true
	} else {
		if len(config.SendHeaders) > 0 {
			http.parserConfig.sendHeaders = true

			http.parserConfig.headersWhitelist = map[string]bool{}
			for _, hdr := range config.SendHeaders {
				http.parserConfig.headersWhitelist[strings.ToLower(hdr)] = true
			}
		}
	}
}

// GetPorts lists the port numbers the HTTP protocol analyser will handle.
func (http *httpPlugin) GetPorts() []int {
	return http.ports
}

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Decides if we can ignore the gap or it's a parser error
// and we need to drop the stream.
func (http *httpPlugin) messageGap(s *stream, nbytes int) (ok bool, complete bool) {
	m := s.message
	switch s.parseState {
	case stateStart, stateHeaders:
		// we know we cannot recover from these
		return false, false
	case stateBody:
		if isDebug {
			debugf("gap in body: %d", nbytes)
		}

		if m.isRequest {
			m.notes = append(m.notes, "Packet loss while capturing the request")
		} else {
			m.notes = append(m.notes, "Packet loss while capturing the response")
		}
		if !m.hasContentLength && (bytes.Equal(m.connection, constClose) ||
			(isVersion(m.version, 1, 0) && !bytes.Equal(m.connection, constKeepAlive))) {
			s.bodyReceived += nbytes
			m.contentLength += nbytes
			return true, false
		} else if len(s.data)+nbytes >= m.contentLength-s.bodyReceived {
			// we're done, but the last portion of the data is gone
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
	st.parseState = stateStart
	st.parseOffset = 0
	st.bodyReceived = 0
	st.message = nil
}

// Called when the parser has identified the boundary
// of a message.
func (http *httpPlugin) messageComplete(
	conn *httpConnectionData,
	tcptuple *common.TCPTuple,
	dir uint8,
	st *stream,
) {
	http.handleHTTP(conn, st.message, tcptuple, dir)
}

// ConnectionTimeout returns the configured HTTP transaction timeout.
func (http *httpPlugin) ConnectionTimeout() time.Duration {
	return http.transactionTimeout
}

// Parse function is used to process TCP payloads.
func (http *httpPlugin) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
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
func (http *httpPlugin) doParse(
	conn *httpConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
) *httpConnectionData {

	if isDetailed {
		detailedf("Payload received: [%s]", pkt.Payload)
	}

	extraMsgSize := 0 // size of a "seen" packet for which we don't store the actual bytes

	st := conn.streams[dir]
	if st == nil {
		st = newStream(pkt, tcptuple)
		conn.streams[dir] = st
	} else {
		// concatenate bytes
		totalLength := len(st.data) + len(pkt.Payload)
		msg := st.message
		if msg != nil {
			totalLength += len(msg.body)
		}
		if totalLength > http.maxMessageSize {
			if isDebug {
				debugf("Stream data too large, ignoring message")
			}
			extraMsgSize = len(pkt.Payload)
		} else {
			st.data = append(st.data, pkt.Payload...)
		}
	}

	for len(st.data) > 0 || extraMsgSize > 0 {
		if st.message == nil {
			st.message = &message{ts: pkt.Ts}
		}

		parser := newParser(&http.parserConfig)
		ok, complete := parser.parse(st, extraMsgSize)
		extraMsgSize = 0
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.streams[dir] = nil
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

func newStream(pkt *protos.Packet, tcptuple *common.TCPTuple) *stream {
	return &stream{
		tcptuple: tcptuple,
		data:     pkt.Payload,
		message:  &message{ts: pkt.Ts},
	}
}

// ReceivedFin will be called when TCP transaction is terminating.
func (http *httpPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	debugf("Received FIN")
	conn := getHTTPConnection(private)
	if conn == nil {
		return private
	}

	stream := conn.streams[dir]
	if stream == nil {
		return conn
	}

	// send whatever data we got so far as complete. This
	// is needed for the HTTP/1.0 without Content-Length situation.
	if stream.message != nil {
		http.handleHTTP(conn, stream.message, tcptuple, dir)

		// and reset message. Probably not needed, just to be sure.
		stream.PrepareForNewMessage()
	}

	return conn
}

// GapInStream is called when a gap of nbytes bytes is found in the stream (due
// to packet loss).
func (http *httpPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(http) exception")

	conn := getHTTPConnection(private)
	if conn == nil {
		return private, false
	}

	stream := conn.streams[dir]
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
		conn.streams[dir] = nil
		return conn, true
	}

	if complete {
		// Current message is complete, we need to publish from here
		http.messageComplete(conn, tcptuple, dir, stream)
	}

	// don't drop the stream, we can ignore the gap
	return private, false
}

func (http *httpPlugin) handleHTTP(
	conn *httpConnectionData,
	m *message,
	tcptuple *common.TCPTuple,
	dir uint8,
) {

	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())
	http.hideHeaders(m)

	if m.isRequest {
		if isDebug {
			debugf("Received request with tuple: %s", m.tcpTuple)
		}
		conn.requests.append(m)
	} else {
		if isDebug {
			debugf("Received response with tuple: %s", m.tcpTuple)
		}
		conn.responses.append(m)
		http.correlate(conn)
	}
}

func (http *httpPlugin) flushResponses(conn *httpConnectionData) {
	for !conn.responses.empty() {
		unmatchedResponses.Add(1)
		resp := conn.responses.pop()
		debugf("Response from unknown transaction: %s. Reporting error.", resp.tcpTuple)
		event := http.newTransaction(nil, resp)
		http.publishTransaction(event)
	}
}

func (http *httpPlugin) flushRequests(conn *httpConnectionData) {
	for !conn.requests.empty() {
		unmatchedRequests.Add(1)
		requ := conn.requests.pop()
		debugf("Request from unknown transaction %s. Reporting error.", requ.tcpTuple)
		event := http.newTransaction(requ, nil)
		http.publishTransaction(event)
	}
}

func (http *httpPlugin) correlate(conn *httpConnectionData) {

	// drop responses with missing requests
	if conn.requests.empty() {
		http.flushResponses(conn)
		return
	}

	// merge requests with responses into transactions
	for !conn.responses.empty() && !conn.requests.empty() {
		requ := conn.requests.pop()
		resp := conn.responses.pop()
		event := http.newTransaction(requ, resp)

		if isDebug {
			debugf("HTTP transaction completed")
		}
		http.publishTransaction(event)
	}
}

func (http *httpPlugin) newTransaction(requ, resp *message) beat.Event {
	status := common.OK_STATUS
	if resp == nil {
		status = common.ERROR_STATUS
		if requ != nil {
			requ.notes = append(requ.notes, "Unmatched request")
		}
	} else if resp.statusCode >= 400 {
		status = common.ERROR_STATUS
	}
	if requ == nil {
		status = common.ERROR_STATUS
		if resp != nil {
			resp.notes = append(resp.notes, "Unmatched response")
		}
	}

	httpDetails := common.MapStr{}
	fields := common.MapStr{
		"type":   "http",
		"status": status,
		"http":   httpDetails,
	}

	var timestamp time.Time

	if requ != nil {
		path, params, err := http.extractParameters(requ)
		if err != nil {
			logp.Warn("Fail to parse HTTP parameters: %v", err)
		}
		httpDetails["request"] = common.MapStr{
			"params":  params,
			"headers": http.collectHeaders(requ),
		}
		fields["method"] = requ.method
		fields["path"] = path
		fields["query"] = fmt.Sprintf("%s %s", requ.method, path)
		fields["bytes_in"] = requ.size

		fields["src"], fields["dst"] = requ.getEndpoints()

		http.setBody(httpDetails["request"].(common.MapStr), requ)

		timestamp = requ.ts

		if len(requ.notes) > 0 {
			fields["notes"] = requ.notes
		}

		if len(requ.realIP) > 0 {
			fields["real_ip"] = requ.realIP
		}

		if http.sendRequest {
			fields["request"] = string(http.makeRawMessage(requ))
		}
	}

	if resp != nil {
		httpDetails["response"] = common.MapStr{
			"code":    resp.statusCode,
			"phrase":  resp.statusPhrase,
			"headers": http.collectHeaders(resp),
		}
		http.setBody(httpDetails["response"].(common.MapStr), resp)
		fields["bytes_out"] = resp.size

		if http.sendResponse {
			fields["response"] = string(http.makeRawMessage(resp))
		}

		if len(resp.notes) > 0 {
			if fields["notes"] != nil {
				fields["notes"] = append(fields["notes"].([]string), resp.notes...)
			} else {
				fields["notes"] = resp.notes
			}
		}
		if requ == nil {
			timestamp = resp.ts
			fields["src"], fields["dst"] = resp.getEndpoints()
		}
	}

	// resp_time in milliseconds
	if requ != nil && resp != nil {
		fields["responsetime"] = int32(resp.ts.Sub(requ.ts).Nanoseconds() / 1e6)
	}

	return beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	}
}

func (http *httpPlugin) makeRawMessage(m *message) string {
	var result []byte
	result = append(result, m.rawHeaders...)
	if m.sendBody {
		result = append(result, m.body...)
	}
	// TODO: (go1.10) Use strings.Builder to avoid allocation/copying
	return string(result)
}

func (http *httpPlugin) publishTransaction(event beat.Event) {
	if http.results == nil {
		return
	}
	http.results(event)
}

func (http *httpPlugin) collectHeaders(m *message) interface{} {
	hdrs := map[string]interface{}{}

	hdrs["content-length"] = m.contentLength
	if len(m.contentType) > 0 {
		hdrs["content-type"] = m.contentType
	}

	if http.parserConfig.sendHeaders {

		cookie := "cookie"
		if !m.isRequest {
			cookie = "set-cookie"
		}

		for name, value := range m.headers {
			if strings.ToLower(name) == "content-type" {
				continue
			}
			if strings.ToLower(name) == "content-length" {
				continue
			}
			if http.splitCookie && name == cookie {
				hdrs[name] = splitCookiesHeader(string(value))
			} else {
				hdrs[name] = value
			}
		}
	}
	return hdrs
}

func (http *httpPlugin) setBody(result common.MapStr, m *message) {
	if m.sendBody && len(m.body) > 0 {
		result["body"] = string(m.body)
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

func (http *httpPlugin) hideHeaders(m *message) {
	if !m.isRequest || !http.redactAuthorization {
		return
	}

	msg := m.rawHeaders
	limit := len(msg)

	// byte64 != encryption, so obscure it in headers in case of Basic Authentication

	redactHeaders := []string{"authorization", "proxy-authorization"}
	authText := []byte("uthorization:") // [aA] case insensitive, also catches Proxy-Authorization:

	authHeaderStartX := m.headerOffset
	authHeaderEndX := limit

	for authHeaderStartX < limit {
		if isDebug {
			debugf("looking for authorization from %d to %d",
				authHeaderStartX, authHeaderEndX)
		}

		startOfHeader := bytes.Index(msg[authHeaderStartX:], authText)
		if startOfHeader >= 0 {
			authHeaderStartX = authHeaderStartX + startOfHeader

			endOfHeader := bytes.Index(msg[authHeaderStartX:], constCRLF)
			if endOfHeader >= 0 {
				authHeaderEndX = authHeaderStartX + endOfHeader

				if authHeaderEndX > limit {
					authHeaderEndX = limit
				}

				if isDebug {
					debugf("Redact authorization from %d to %d", authHeaderStartX, authHeaderEndX)
				}

				for i := authHeaderStartX + len(authText); i < authHeaderEndX; i++ {
					msg[i] = byte('*')
				}
			}
		}
		authHeaderStartX = authHeaderEndX + len(constCRLF)
		authHeaderEndX = len(m.rawHeaders)
	}

	for _, header := range redactHeaders {
		if len(m.headers[header]) > 0 {
			m.headers[header] = []byte("*")
		}
	}
}

func (http *httpPlugin) hideSecrets(values url.Values) url.Values {
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
func (http *httpPlugin) extractParameters(m *message) (path string, params string, err error) {
	var values url.Values

	u, err := url.Parse(string(m.requestURI))
	if err != nil {
		return
	}
	values = u.Query()
	path = u.Path

	paramsMap := http.hideSecrets(values)

	if m.contentLength > 0 && m.saveBody && bytes.Contains(m.contentType, []byte("urlencoded")) {

		values, err = url.ParseQuery(string(m.body))
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

func (http *httpPlugin) isSecretParameter(key string) bool {
	for _, keyword := range http.hideKeywords {
		if strings.ToLower(key) == keyword {
			return true
		}
	}
	return false
}

func (http *httpPlugin) Expired(tuple *common.TCPTuple, private protos.ProtocolData) {
	conn := getHTTPConnection(private)
	if conn == nil {
		return
	}
	if isDebug {
		debugf("expired connection %s", tuple)
	}
	// terminate streams
	for dir, s := range conn.streams {
		// Do not send incomplete or empty messages
		if s != nil && s.message != nil && s.message.headersReceived() {
			if isDebug {
				debugf("got message %+v", s.message)
			}
			http.handleHTTP(conn, s.message, tuple, uint8(dir))
			s.PrepareForNewMessage()
		}
	}
	// correlate transactions
	http.correlate(conn)

	// flush uncorrelated requests and responses
	http.flushRequests(conn)
	http.flushResponses(conn)
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
