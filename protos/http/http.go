package http

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"

	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/procs"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/tcp"
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

type HttpStream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset  int
	parseState   parserState
	bodyReceived int

	message *HttpMessage
}

type httpPrivateData struct {
	Data [2]*HttpStream
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

func (http *Http) getTransaction(k common.HashableTcpTuple) *HttpTransaction {
	v := http.transactions.Get(k)
	if v != nil {
		return v.(*HttpTransaction)
	}
	return nil
}

func (http *Http) InitDefaults() {
	http.SendRequest = false
	http.SendResponse = false
	http.RedactAuthorization = false
	http.transactionTimeout = protos.DefaultTransactionExpiration
}

func (http *Http) SetFromConfig(config config.Http) (err error) {

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

// messageGap is called when a gap of size `nbytes` is found in the
// tcp stream. Decides if we can ignore the gap or it's a parser error
// and we need to drop the stream.
func (http *Http) messageGap(s *HttpStream, nbytes int) (ok bool, complete bool) {

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

func (stream *HttpStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.message.end:]
	stream.parseState = stateStart
	stream.parseOffset = 0
	stream.bodyReceived = 0
	stream.message = nil
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

	detailedf("Payload received: [%s]", pkt.Payload)

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
			debugf("Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}
	stream := priv.Data[dir]
	if stream.message == nil {
		stream.message = &HttpMessage{Ts: pkt.Ts}
	}

	parser := newParser(&http.parserConfig)
	ok, complete := parser.parse(stream)
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

		detailedf("Publish something on connection FIN")

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
	detailedf("messageGap returned ok=%v complete=%v", ok, complete)
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

	debugf("Received request with tuple: %s", msg.TcpTuple)

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
	if http.SendRequest {
		trans.Request_raw = string(http.cutMessageBody(msg))
	}

	trans.Method = msg.Method
	trans.RequestUri = msg.RequestUri
	trans.BytesIn = msg.Size
	trans.Notes = msg.Notes

	trans.Http = common.MapStr{}

	if http.parserConfig.SendHeaders {
		if !http.SplitCookie {
			trans.Http["request_headers"] = msg.Headers
		} else {
			hdrs := common.MapStr{}
			for name, value := range msg.Headers {
				if name == "cookie" {
					hdrs[name] = splitCookiesHeader(value)
				} else {
					hdrs[name] = value
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

	debugf("Received response with tuple: %s", tuple)

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
	trans.Http.Update(response)
	trans.Notes = append(trans.Notes, msg.Notes...)

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	// save Raw message
	if http.SendResponse {
		trans.Response_raw = string(http.cutMessageBody(msg))
	}

	http.publishTransaction(trans)
	http.transactions.Delete(trans.tuple.Hashable())

	debugf("HTTP transaction completed: %s\n", trans.Http)
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
	if http.SendRequest {
		event["request"] = t.Request_raw
	}
	if http.SendResponse {
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

func (http *Http) cutMessageBody(m *HttpMessage) []byte {
	cutMsg := []byte{}

	// add headers always
	cutMsg = m.Raw[:m.bodyOffset]

	// add body
	if len(m.ContentType) == 0 || http.shouldIncludeInBody(m.ContentType) {
		if len(m.chunked_body) > 0 {
			cutMsg = append(cutMsg, m.chunked_body...)
		} else {
			debugf("Body to include: [%s]", m.Raw[m.bodyOffset:])
			cutMsg = append(cutMsg, m.Raw[m.bodyOffset:]...)
		}
	}

	return cutMsg
}

func (http *Http) shouldIncludeInBody(contenttype string) bool {
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

func (http *Http) hideHeaders(m *HttpMessage, msg []byte) {

	if m.IsRequest {
		// byte64 != encryption, so obscure it in headers in case of Basic Authentication
		if http.RedactAuthorization {

			redactHeaders := []string{"authorization", "proxy-authorization"}
			auth_text := []byte("uthorization:") // [aA] case insensitive, also catches Proxy-Authorization:

			authHeaderStartX := m.headerOffset
			authHeaderEndX := m.bodyOffset

			for authHeaderStartX < m.bodyOffset {
				debugf("looking for authorization from %d to %d", authHeaderStartX, authHeaderEndX)

				startOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], auth_text)
				if startOfHeader >= 0 {
					authHeaderStartX = authHeaderStartX + startOfHeader

					endOfHeader := bytes.Index(msg[authHeaderStartX:m.bodyOffset], []byte("\r\n"))
					if endOfHeader >= 0 {
						authHeaderEndX = authHeaderStartX + endOfHeader

						if authHeaderEndX > m.bodyOffset {
							authHeaderEndX = m.bodyOffset
						}

						debugf("Redact authorization from %d to %d", authHeaderStartX, authHeaderEndX)

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

	detailedf("Parameters: %s", params)

	return
}

func (http *Http) isSecretParameter(key string) bool {

	for _, keyword := range http.HideKeywords {
		if strings.ToLower(key) == keyword {
			return true
		}
	}
	return false
}
