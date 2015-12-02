package http

import (
	"bytes"
	"fmt"
	"net/url"
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

type stream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset  int
	parseState   parserState
	bodyReceived int

	message *message
}

type httpConnectionData struct {
	Streams [2]*stream
}

type transaction struct {
	Type         string
	tuple        common.TcpTuple
	Src          common.Endpoint
	Dst          common.Endpoint
	RealIP       string
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	cmdline      *common.CmdlineTuple
	Method       string
	RequestURI   string
	Params       string
	Path         string
	BytesOut     uint64
	BytesIn      uint64
	Notes        []string

	HTTP common.MapStr

	RequestRaw  string
	ResponseRaw string
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

	parserConfig parserConfig

	transactions       *common.Cache
	transactionTimeout time.Duration

	results publisher.Client
}

func (http *HTTP) getTransaction(k common.HashableTcpTuple) *transaction {
	v := http.transactions.Get(k)
	if v != nil {
		return v.(*transaction)
	}
	return nil
}

func (http *HTTP) initDefaults() {
	http.SendRequest = false
	http.SendResponse = false
	http.RedactAuthorization = false
	http.transactionTimeout = protos.DefaultTransactionExpiration
}

func (http *HTTP) setFromConfig(config config.Http) (err error) {

	http.Ports = config.Ports

	if config.SendRequest != nil {
		http.SendRequest = *config.SendRequest
	}
	if config.SendResponse != nil {
		http.SendResponse = *config.SendResponse
	}
	http.HideKeywords = config.Hide_keywords
	if config.Redact_authorization != nil {
		http.RedactAuthorization = *config.Redact_authorization
	}

	if config.Send_all_headers != nil {
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

	if config.Split_cookie != nil {
		http.SplitCookie = *config.Split_cookie
	}

	if config.Real_ip_header != nil {
		http.parserConfig.RealIPHeader = strings.ToLower(*config.Real_ip_header)
	}

	if config.TransactionTimeout != nil && *config.TransactionTimeout > 0 {
		http.transactionTimeout = time.Duration(*config.TransactionTimeout) * time.Second
	}

	return nil
}

// GetPorts lists the port numbers the HTTP protocol analyser will handle.
func (http *HTTP) GetPorts() []int {
	return http.Ports
}

// Init initializes the HTTP protocol analyser.
func (http *HTTP) Init(testMode bool, results publisher.Client) error {
	http.initDefaults()

	if !testMode {
		err := http.setFromConfig(config.ConfigSingleton.Protocols.Http)
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
		debugf("gap in body: %d", nbytes)
		if m.IsRequest {
			m.Notes = append(m.Notes, "Packet loss while capturing the request")
		} else {
			m.Notes = append(m.Notes, "Packet loss while capturing the response")
		}
		if !m.hasContentLength && (m.connection == "close" ||
			(isVersion(m.version, 1, 0) && m.connection != "keep-alive")) {

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
func (http *HTTP) messageComplete(tcptuple *common.TcpTuple, dir uint8, st *stream) {
	msg := st.data[st.message.start:st.message.end]
	http.hideHeaders(st.message, msg)

	http.handleHTTP(st.message, tcptuple, dir, msg)

	// and reset message
	st.PrepareForNewMessage()
}

// ConnectionTimeout returns the configured HTTP transaction timeout.
func (http *HTTP) ConnectionTimeout() time.Duration {
	return http.transactionTimeout
}

// Parse function is used to process TCP payloads.
func (http *HTTP) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParseHttp exception")

	detailedf("Payload received: [%s]", pkt.Payload)

	priv := httpConnectionData{}
	if private != nil {
		var ok bool
		priv, ok = private.(httpConnectionData)
		if !ok {
			priv = httpConnectionData{}
		}
	}

	if priv.Streams[dir] == nil {
		priv.Streams[dir] = &stream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &message{Ts: pkt.Ts},
		}

	} else {
		// concatenate bytes
		priv.Streams[dir].data = append(priv.Streams[dir].data, pkt.Payload...)
		if len(priv.Streams[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			debugf("Stream data too large, dropping TCP stream")
			priv.Streams[dir] = nil
			return priv
		}
	}
	stream := priv.Streams[dir]
	if stream.message == nil {
		stream.message = &message{Ts: pkt.Ts}
	}

	parser := newParser(&http.parserConfig)
	ok, complete := parser.parse(stream)
	if !ok {
		// drop this tcp stream. Will retry parsing with the next
		// segment in it
		priv.Streams[dir] = nil
		return priv
	}

	if complete {
		// all ok, ship it
		http.messageComplete(tcptuple, dir, stream)
	}

	return priv
}

// ReceivedFin will be called when TCP transaction is terminating.
func (http *HTTP) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	if private == nil {
		return private
	}
	httpData, ok := private.(httpConnectionData)
	if !ok {
		return private
	}
	if httpData.Streams[dir] == nil {
		return httpData
	}

	stream := httpData.Streams[dir]

	// send whatever data we got so far as complete. This
	// is needed for the HTTP/1.0 without Content-Length situation.
	if stream.message != nil &&
		len(stream.data[stream.message.start:]) > 0 {

		detailedf("Publish something on connection FIN")

		msg := stream.data[stream.message.start:]
		http.hideHeaders(stream.message, msg)

		http.handleHTTP(stream.message, tcptuple, dir, msg)

		// and reset message. Probably not needed, just to be sure.
		stream.PrepareForNewMessage()
	}

	return httpData
}

// GapInStream is called when a gap of nbytes bytes is found in the stream (due
// to packet loss).
func (http *HTTP) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	defer logp.Recover("GapInStream(http) exception")

	if private == nil {
		return private, false
	}
	httpData, ok := private.(httpConnectionData)
	if !ok {
		return private, false
	}
	stream := httpData.Streams[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	ok, complete := http.messageGap(stream, nbytes)
	detailedf("messageGap returned ok=%v complete=%v", ok, complete)
	if !ok {
		// on errors, drop stream
		httpData.Streams[dir] = nil
		return httpData, true
	}

	if complete {
		// Current message is complete, we need to publish from here
		http.messageComplete(tcptuple, dir, stream)
	}

	// don't drop the stream, we can ignore the gap
	return private, false
}

func (http *HTTP) handleHTTP(m *message, tcptuple *common.TcpTuple,
	dir uint8, rawMsg []byte) {

	m.TCPTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())
	m.Raw = rawMsg

	if m.IsRequest {
		http.receivedHTTPRequest(m)
	} else {
		http.receivedHTTPResponse(m)
	}
}

func (http *HTTP) receivedHTTPRequest(msg *message) {

	trans := http.getTransaction(msg.TCPTuple.Hashable())
	if trans != nil {
		if len(trans.HTTP) != 0 {
			logp.Warn("Two requests without a response. Dropping old request")
		}
	} else {
		trans = &transaction{Type: "http", tuple: msg.TCPTuple}
		http.transactions.Put(msg.TCPTuple.Hashable(), trans)
	}

	debugf("Received request with tuple: %s", msg.TCPTuple)

	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000)
	trans.JsTs = msg.Ts
	trans.Src = common.Endpoint{
		Ip:   msg.TCPTuple.Src_ip.String(),
		Port: msg.TCPTuple.Src_port,
		Proc: string(msg.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   msg.TCPTuple.Dst_ip.String(),
		Port: msg.TCPTuple.Dst_port,
		Proc: string(msg.CmdlineTuple.Dst),
	}
	if msg.Direction == tcp.TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}

	// save Raw message
	if http.SendRequest {
		trans.RequestRaw = string(http.cutMessageBody(msg))
	}

	trans.Method = msg.Method
	trans.RequestURI = msg.RequestURI
	trans.BytesIn = msg.Size
	trans.Notes = msg.Notes

	trans.HTTP = common.MapStr{}

	if http.parserConfig.SendHeaders {
		if !http.SplitCookie {
			trans.HTTP["request_headers"] = msg.Headers
		} else {
			hdrs := common.MapStr{}
			for name, value := range msg.Headers {
				if name == "cookie" {
					hdrs[name] = splitCookiesHeader(value)
				} else {
					hdrs[name] = value
				}
			}

			trans.HTTP["request_headers"] = hdrs
		}
	}

	trans.RealIP = msg.RealIP

	var err error
	trans.Path, trans.Params, err = http.extractParameters(msg, msg.Raw)
	if err != nil {
		logp.Warn("http", "Fail to parse HTTP parameters: %v", err)
	}
}

func (http *HTTP) receivedHTTPResponse(msg *message) {

	// we need to search the request first.
	tuple := msg.TCPTuple

	debugf("Received response with tuple: %s", tuple)

	trans := http.getTransaction(tuple.Hashable())
	if trans == nil {
		logp.Warn("Response from unknown transaction. Ignoring: %v", tuple)
		return
	}

	if trans.HTTP == nil {
		logp.Warn("Response without a known request. Ignoring.")
		return
	}

	response := common.MapStr{
		"phrase":         msg.StatusPhrase,
		"code":           msg.StatusCode,
		"content_length": msg.ContentLength,
	}

	if http.parserConfig.SendHeaders {
		if !http.SplitCookie {
			response["response_headers"] = msg.Headers
		} else {
			hdrs := common.MapStr{}
			for name, value := range msg.Headers {
				if name == "set-cookie" {
					hdrs[name] = splitCookiesHeader(value)
				} else {
					hdrs[name] = value
				}
			}

			response["response_headers"] = hdrs
		}
	}

	trans.BytesOut = msg.Size
	trans.HTTP.Update(response)
	trans.Notes = append(trans.Notes, msg.Notes...)

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if http.SendResponse {
		trans.ResponseRaw = string(http.cutMessageBody(msg))
	}

	http.publishTransaction(trans)
	http.transactions.Delete(trans.tuple.Hashable())

	debugf("HTTP transaction completed: %s\n", trans.HTTP)
}

func (http *HTTP) publishTransaction(t *transaction) {

	if http.results == nil {
		return
	}

	event := common.MapStr{}

	event["type"] = "http"
	code := t.HTTP["code"].(uint16)
	if code < 400 {
		event["status"] = common.OK_STATUS
	} else {
		event["status"] = common.ERROR_STATUS
	}
	event["responsetime"] = t.ResponseTime
	if http.SendRequest {
		event["request"] = t.RequestRaw
	}
	if http.SendResponse {
		event["response"] = t.ResponseRaw
	}
	event["http"] = t.HTTP
	if len(t.RealIP) > 0 {
		event["real_ip"] = t.RealIP
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

func (http *HTTP) cutMessageBody(m *message) []byte {
	cutMsg := []byte{}

	// add headers always
	cutMsg = m.Raw[:m.bodyOffset]

	// add body
	if len(m.ContentType) == 0 || http.shouldIncludeInBody(m.ContentType) {
		if len(m.chunkedBody) > 0 {
			cutMsg = append(cutMsg, m.chunkedBody...)
		} else {
			debugf("Body to include: [%s]", m.Raw[m.bodyOffset:])
			cutMsg = append(cutMsg, m.Raw[m.bodyOffset:]...)
		}
	}

	return cutMsg
}

func (http *HTTP) shouldIncludeInBody(contenttype string) bool {
	includedBodies := config.ConfigSingleton.Protocols.Http.Include_body_for
	for _, include := range includedBodies {
		if strings.Contains(contenttype, include) {
			debugf("Should Include Body = true Content-Type " + contenttype + " include_body " + include)
			return true
		}
		debugf("Should Include Body = false Content-Type" + contenttype + " include_body " + include)
	}
	return false
}

func (http *HTTP) hideHeaders(m *message, msg []byte) {

	if m.IsRequest {
		// byte64 != encryption, so obscure it in headers in case of Basic Authentication
		if http.RedactAuthorization {

			redactHeaders := []string{"authorization", "proxy-authorization"}
			authText := []byte("uthorization:") // [aA] case insensitive, also catches Proxy-Authorization:

			authHeaderStartX := m.headerOffset
			authHeaderEndX := m.bodyOffset

			for authHeaderStartX < m.bodyOffset {
				debugf("looking for authorization from %d to %d", authHeaderStartX, authHeaderEndX)

				startOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], authText)
				if startOfHeader >= 0 {
					authHeaderStartX = authHeaderStartX + startOfHeader

					endOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], []byte("\r\n"))
					if endOfHeader >= 0 {
						authHeaderEndX = authHeaderStartX + endOfHeader

						if authHeaderEndX > m.bodyOffset {
							authHeaderEndX = m.bodyOffset
						}

						debugf("Redact authorization from %d to %d", authHeaderStartX, authHeaderEndX)

						for i := authHeaderStartX + len(authText); i < authHeaderEndX; i++ {
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
// Returns the Request URI path and the (ajdusted) parameters.
func (http *HTTP) extractParameters(m *message, msg []byte) (path string, params string, err error) {
	var values url.Values

	u, err := url.Parse(m.RequestURI)
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

	detailedf("Parameters: %s", params)

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
