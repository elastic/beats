// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package http

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	debugf    = logp.MakeDebug("http")
	detailedf = logp.MakeDebug("httpdetailed")
)

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
	redactHeaders       []string
	maxMessageSize      int
	mustDecodeBody      bool

	parserConfig parserConfig

	transactionTimeout time.Duration

	results protos.Reporter
	watcher procs.ProcessesWatcher
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
	watcher procs.ProcessesWatcher,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &httpPlugin{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, watcher, &config); err != nil {
		return nil, err
	}
	return p, nil
}

// Init initializes the HTTP protocol analyser.
func (http *httpPlugin) init(results protos.Reporter, watcher procs.ProcessesWatcher, config *httpConfig) error {
	http.setFromConfig(config)

	isDebug = logp.IsDebug("http")
	isDetailed = logp.IsDebug("httpdetailed")
	http.results = results
	http.watcher = watcher
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
	http.mustDecodeBody = config.DecodeBody

	http.redactHeaders = make([]string, len(config.RedactHeaders))
	for i, header := range config.RedactHeaders {
		http.redactHeaders[i] = strings.ToLower(header)
	}

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
			if !m.packetLossReq {
				m.packetLossReq = true
				m.notes = append(m.notes, "Packet loss while capturing the request")
			}
		} else {
			if !m.packetLossResp {
				m.packetLossResp = true
				m.notes = append(m.notes, "Packet loss while capturing the response")
			}
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
	private protos.ProtocolData,
) protos.ProtocolData {
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
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool,
) {
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
	m.cmdlineTuple = http.watcher.FindProcessesTupleTCP(tcptuple.IPPort())

	if !http.redactAuthorization {
		m.username = extractBasicAuthUser(m.headers)
	}

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

		if resp.statusCode == 100 {
			debugf("Drop first 100-continue response")
			return
		}

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

	var ts time.Time
	var src, dst *common.Endpoint
	for _, m := range []*message{requ, resp} {
		if m == nil {
			continue
		}
		ts = m.ts
		src, dst = m.getEndpoints()
		break
	}

	evt, pbf := pb.NewBeatEvent(ts)
	pbf.SetSource(src)
	pbf.SetDestination(dst)
	pbf.AddIP(src.IP)
	pbf.AddIP(dst.IP)
	pbf.Network.Transport = "tcp"
	pbf.Network.Protocol = "http"

	fields := evt.Fields
	fields["type"] = pbf.Network.Protocol
	fields["status"] = status

	var httpFields ProtocolFields
	if requ != nil {
		http.decodeBody(requ)
		path, params, err := http.extractParameters(requ)
		if err != nil {
			logp.Warn("Fail to parse HTTP parameters: %v", err)
		}

		pbf.Source.Bytes = int64(requ.size)
		host, port := extractHostHeader(string(requ.host))
		if net.ParseIP(host) == nil {
			pbf.Destination.Domain = host
			pbf.AddHost(host)
		} else {
			pbf.AddIP(host)
		}
		if port == 0 {
			port = int(pbf.Destination.Port)
		} else if port != int(pbf.Destination.Port) {
			requ.notes = append(requ.notes, "Host header port number mismatch")
		}
		pbf.Event.Start = requ.ts
		pbf.Network.ForwardedIP = string(requ.realIP)
		pbf.AddIP(string(requ.realIP))
		pbf.Error.Message = requ.notes

		// http
		httpFields.Version = requ.version.String()
		httpFields.RequestBytes = int64(requ.size)
		httpFields.RequestBodyBytes = int64(requ.contentLength)
		httpFields.RequestMethod = requ.method
		httpFields.RequestReferrer = requ.referer
		pbf.AddHost(string(requ.referer))
		if requ.sendBody && len(requ.body) > 0 {
			httpFields.RequestBodyBytes = int64(len(requ.body))
			httpFields.RequestBodyContent = common.NetString(requ.body)
		}
		httpFields.RequestHeaders = http.collectHeaders(requ)

		// url
		u := newURL(host, int64(port), path, params)
		pb.MarshalStruct(evt.Fields, "url", u)

		// user-agent
		userAgent := ecs.UserAgent{Original: string(requ.userAgent)}
		pb.MarshalStruct(evt.Fields, "user_agent", userAgent)

		// packetbeat root fields
		if http.sendRequest {
			fields["request"] = string(http.makeRawMessage(requ))
		}
		fields["method"] = httpFields.RequestMethod
		fields["query"] = fmt.Sprintf("%s %s", requ.method, path)

		if requ.username != "" {
			fields["user.name"] = requ.username
			pbf.AddUser(requ.username)
		}
	}

	if resp != nil {
		http.decodeBody(resp)

		pbf.Destination.Bytes = int64(resp.size)
		pbf.Event.End = resp.ts
		pbf.Error.Message = append(pbf.Error.Message, resp.notes...)

		// http
		httpFields.ResponseStatusCode = int64(resp.statusCode)
		httpFields.ResponseStatusPhrase = bytes.ToLower(resp.statusPhrase)
		httpFields.ResponseBytes = int64(resp.size)
		httpFields.ResponseBodyBytes = int64(resp.contentLength)
		if resp.sendBody && len(resp.body) > 0 {
			httpFields.ResponseBodyBytes = int64(len(resp.body))
			httpFields.ResponseBodyContent = common.NetString(resp.body)
		}
		httpFields.ResponseHeaders = http.collectHeaders(resp)

		// packetbeat root fields
		if http.sendResponse {
			fields["response"] = string(http.makeRawMessage(resp))
		}
	}

	pb.MarshalStruct(evt.Fields, "http", httpFields)
	return evt
}

func (http *httpPlugin) makeRawMessage(m *message) string {
	if m.sendBody {
		var b strings.Builder
		b.Grow(len(m.rawHeaders) + len(m.body))
		b.Write(m.rawHeaders)
		b.Write(m.body)
		return b.String()
	}
	return string(m.rawHeaders)
}

func (http *httpPlugin) publishTransaction(event beat.Event) {
	if http.results == nil {
		return
	}
	http.results(event)
}

func (http *httpPlugin) collectHeaders(m *message) mapstr.M {
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
			switch {
			case bytes.Equal([]byte(name), nameContentLength),
				bytes.Equal([]byte(name), nameContentType):
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

func (http *httpPlugin) decodeBody(m *message) {
	if m.saveBody && len(m.body) > 0 {
		if http.mustDecodeBody && len(m.encodings) > 0 {
			var err error
			m.body, err = decodeBody(m.body, m.encodings, http.maxMessageSize)
			if err != nil {
				// Body can contain partial data
				m.notes = append(m.notes, err.Error())
			}
		}
	}
}

func decodeBody(body []byte, encodings []string, maxSize int) (result []byte, err error) {
	if isDebug {
		debugf("decoding body with encodings=%v", encodings)
	}
	for idx := len(encodings) - 1; idx >= 0; idx-- {
		format := encodings[idx]
		body, err = decodeHTTPBody(body, format, maxSize)
		if err != nil {
			// Do not output a partial body unless failure occurs on the
			// last decoder.
			if idx != 0 {
				body = nil
			}
			return body, errors.Wrapf(err, "unable to decode body using %s encoding", format)
		}
	}
	return body, nil
}

func splitCookiesHeader(headerVal string) map[string]string {
	cookies := map[string]string{}

	cstring := strings.Split(headerVal, ";")
	for _, cval := range cstring {
		cookie := strings.SplitN(cval, "=", 2)
		if len(cookie) == 2 {
			cookies[strings.ToLower(strings.TrimSpace(cookie[0]))] = parseCookieValue(strings.TrimSpace(cookie[1]))
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

func extractHostHeader(header string) (host string, port int) {
	if len(header) == 0 || net.ParseIP(header) != nil {
		return header, port
	}
	// Split :port trailer
	if pos := strings.LastIndexByte(header, ':'); pos != -1 {
		if num, err := strconv.Atoi(header[pos+1:]); err == nil && num > 0 && num < 65536 {
			header, port = header[:pos], num
		}
	}
	// Remove square bracket boxing of IPv6 address.
	if last := len(header) - 1; header[0] == '[' && header[last] == ']' && net.ParseIP(header[1:last]) != nil {
		header = header[1:last]
	}
	return header, port
}

func (http *httpPlugin) hideHeaders(m *message) {
	for _, header := range http.redactHeaders {
		if _, exists := m.headers[header]; exists {
			m.headers[header] = []byte("REDACTED")
		}
	}

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

func extractBasicAuthUser(headers map[string]common.NetString) string {
	const prefix = "Basic "

	auth := string(headers["authorization"])
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return ""
	}

	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		c, err = base64.RawStdEncoding.DecodeString(auth[len(prefix):])
		if err != nil {
			return ""
		}
	}

	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return ""
	}

	return cs[:s]
}
