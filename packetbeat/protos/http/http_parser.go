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
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/streambuf"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/packetbeat/protos/tcp"
)

// Http Message
type message struct {
	ts               time.Time
	hasContentLength bool
	headerOffset     int
	version          version
	connection       common.NetString
	chunkedLength    int

	isRequest    bool
	tcpTuple     common.TCPTuple
	cmdlineTuple *common.ProcessTuple
	direction    uint8

	// Request Info
	requestURI   common.NetString
	method       common.NetString
	statusCode   uint16
	statusPhrase common.NetString
	realIP       common.NetString

	// Http Headers
	contentLength int
	contentType   common.NetString
	host          common.NetString
	referer       common.NetString
	userAgent     common.NetString
	encodings     []string
	isChunked     bool
	headers       map[string]common.NetString
	size          uint64
	username      string

	rawHeaders []byte

	// sendBody determines if the body must be sent along with the event
	// because the content-type is included in the send_body_for setting.
	sendBody bool
	// saveBody determines if the body must be saved. It is set when sendBody
	// is true or when the body type is form-urlencoded.
	saveBody bool
	body     []byte

	notes          []string
	packetLossReq  bool
	packetLossResp bool

	next *message
}

type version struct {
	major uint8
	minor uint8
}

func (v version) String() string {
	if v.major == 1 && v.minor == 1 {
		return "1.1"
	}
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

type parser struct {
	config *parserConfig
}

type parserConfig struct {
	realIPHeader           string
	sendHeaders            bool
	sendAllHeaders         bool
	headersWhitelist       map[string]bool
	includeRequestBodyFor  []string
	includeResponseBodyFor []string
}

var (
	transferEncodingChunked = "chunked"

	constCRLF = []byte("\r\n")

	constClose       = []byte("close")
	constKeepAlive   = []byte("keep-alive")
	constHTTPVersion = []byte("HTTP/")

	nameContentLength    = []byte("content-length")
	nameContentType      = []byte("content-type")
	nameTransferEncoding = []byte("transfer-encoding")
	nameContentEncoding  = []byte("content-encoding")
	nameConnection       = []byte("connection")
	nameHost             = []byte("host")
	nameReferer          = []byte("referer")
	nameUserAgent        = []byte("user-agent")
)

func newParser(config *parserConfig) *parser {
	return &parser{config: config}
}

func (parser *parser) parse(s *stream, extraMsgSize int) (bool, bool) {
	m := s.message

	if extraMsgSize > 0 {
		// A packet of extraMsgSize size was seen, but we don't have
		// its actual bytes. This is only usable in the `stateBody` state.
		if s.parseState != stateBody {
			return false, false
		}
		return parser.eatBody(s, m, extraMsgSize)
	}

	for s.parseOffset < len(s.data) {
		switch s.parseState {
		case stateStart:
			if cont, ok, complete := parser.parseHTTPLine(s, m); !cont {
				return ok, complete
			}
		case stateHeaders:
			if cont, ok, complete := parser.parseHeaders(s, m); !cont {
				return ok, complete
			}
		case stateBody:
			return parser.parseBody(s, m)
		case stateBodyChunkedStart:
			if cont, ok, complete := parser.parseBodyChunkedStart(s, m); !cont {
				return ok, complete
			}
		case stateBodyChunked:
			if cont, ok, complete := parser.parseBodyChunked(s, m); !cont {
				return ok, complete
			}
		case stateBodyChunkedWaitFinalCRLF:
			return parser.parseBodyChunkedWaitFinalCRLF(s, m)
		}
	}

	return true, false
}

func (*parser) parseHTTPLine(s *stream, m *message) (cont, ok, complete bool) {
	i := bytes.Index(s.data[s.parseOffset:], []byte("\r\n"))
	if i == -1 {
		return false, true, false
	}

	// Very basic tests on the first line. Just to check that
	// we have what looks as an HTTP message
	var version []byte
	var err error
	fline := s.data[s.parseOffset:i]
	if len(fline) < 9 {
		if isDebug {
			debugf("First line too small")
		}
		return false, false, false
	}
	if bytes.Equal(fline[0:5], constHTTPVersion) {
		// RESPONSE
		m.isRequest = false
		version = fline[5:8]
		m.statusCode, m.statusPhrase, err = parseResponseStatus(fline[9:])
		if err != nil {
			logp.Warn("Failed to understand HTTP response status: %s", fline[9:])
			return false, false, false
		}

		if isDebug {
			debugf("HTTP status_code=%d, status_phrase=%s", m.statusCode, m.statusPhrase)
		}
	} else {
		// REQUEST
		afterMethodIdx := bytes.IndexFunc(fline, unicode.IsSpace)
		afterRequestURIIdx := bytes.LastIndexFunc(fline, unicode.IsSpace)

		// Make sure we have the VERB + URI + HTTP_VERSION
		if afterMethodIdx == -1 || afterRequestURIIdx == -1 || afterMethodIdx == afterRequestURIIdx {
			if isDebug {
				debugf("Couldn't understand HTTP request: %s", fline)
			}
			return false, false, false
		}

		m.method = common.NetString(fline[:afterMethodIdx])
		m.requestURI = common.NetString(fline[afterMethodIdx+1 : afterRequestURIIdx])

		versionIdx := afterRequestURIIdx + len(constHTTPVersion) + 1
		if len(fline) > versionIdx && bytes.Equal(fline[afterRequestURIIdx+1:versionIdx], constHTTPVersion) {
			m.isRequest = true
			version = fline[versionIdx:]
		} else {
			if isDebug {
				debugf("Couldn't understand HTTP version: %s", fline)
			}
			return false, false, false
		}
	}

	m.version.major, m.version.minor, err = parseVersion(version)
	if err != nil {
		if isDebug {
			debugf("Failed to understand HTTP version: %v", version)
		}
		m.version.major = 1
		m.version.minor = 0
	}
	if isDebug {
		debugf("HTTP version %d.%d", m.version.major, m.version.minor)
	}

	// ok so far
	s.parseOffset = i + 2
	m.headerOffset = s.parseOffset
	s.parseState = stateHeaders

	return true, true, true
}

func parseResponseStatus(s []byte) (uint16, []byte, error) {
	if isDebug {
		debugf("parseResponseStatus: %s", s)
	}

	var phrase []byte
	p := bytes.IndexByte(s, ' ')
	if p == -1 {
		p = len(s)
	} else {
		phrase = s[p+1:]
	}
	statusCode, err := parseInt(s[0:p])
	if err != nil {
		return 0, nil, fmt.Errorf("Unable to parse status code from [%s]", s)
	}
	return uint16(statusCode), phrase, nil
}

func parseVersion(s []byte) (uint8, uint8, error) {
	if len(s) < 3 {
		return 0, 0, errors.New("Invalid version")
	}

	major := s[0] - '0'
	minor := s[2] - '0'
	if major > 1 || minor > 2 {
		return 0, 0, errors.New("unsupported version")
	}
	return uint8(major), uint8(minor), nil
}

func (parser *parser) parseHeaders(s *stream, m *message) (cont, ok, complete bool) {
	if len(s.data)-s.parseOffset >= 2 &&
		bytes.Equal(s.data[s.parseOffset:s.parseOffset+2], []byte("\r\n")) {
		// EOH
		m.size = uint64(s.parseOffset + 2)
		m.rawHeaders = s.data[:m.size]
		s.data = s.data[m.size:]
		s.parseOffset = 0

		if !m.isRequest && ((100 <= m.statusCode && m.statusCode < 200) || m.statusCode == 204 || m.statusCode == 304) {
			// response with a 1xx, 204 , or 304 status  code is always terminated
			// by the first empty line after the  header fields
			if isDebug {
				debugf("Terminate response, status code %d", m.statusCode)
			}
			return false, true, true
		}

		if m.isRequest {
			m.sendBody = parser.shouldIncludeInBody(m.contentType, parser.config.includeRequestBodyFor)
		} else {
			m.sendBody = parser.shouldIncludeInBody(m.contentType, parser.config.includeResponseBodyFor)
		}
		m.saveBody = m.sendBody || (m.contentLength > 0 && bytes.Contains(m.contentType, []byte("urlencoded")))

		if m.isChunked {
			// support for HTTP/1.1 Chunked transfer
			// Transfer-Encoding overrides the Content-Length
			if isDebug {
				debugf("Read chunked body")
			}
			s.parseState = stateBodyChunkedStart
			return true, true, true
		}

		if m.contentLength == 0 && (m.isRequest || m.hasContentLength) {
			if isDebug {
				debugf("Empty content length, ignore body")
			}
			// Ignore body for request that contains a message body but not a Content-Length
			return false, true, true
		}

		if isDebug {
			debugf("Read body")
		}
		s.parseState = stateBody
	} else {
		ok, hfcomplete, offset := parser.parseHeader(m, s.data[s.parseOffset:])
		if !ok {
			return false, false, false
		}
		if !hfcomplete {
			return false, true, false
		}
		s.parseOffset += offset
	}
	return true, true, true
}

func (parser *parser) parseHeader(m *message, data []byte) (bool, bool, int) {
	if m.headers == nil {
		m.headers = make(map[string]common.NetString)
	}
	i := bytes.Index(data, []byte(":"))
	if i == -1 {
		// Expected \":\" in headers. Assuming incomplete"
		return true, false, 0
	}

	config := parser.config

	// enabled if required. Allocs for parameters slow down parser big times
	if isDetailed {
		detailedf("Data: %s", data)
		detailedf("Header: %s", data[:i])
	}

	// skip folding line
	for p := i + 1; p < len(data); {
		q := bytes.Index(data[p:], constCRLF)
		if q == -1 {
			// Assuming incomplete
			return true, false, 0
		}
		p += q
		if len(data) > p && (data[p+1] == ' ' || data[p+1] == '\t') {
			p = p + 2
		} else {
			var headerNameBuf [140]byte
			headerName := toLower(headerNameBuf[:], data[:i])
			headerVal := trim(data[i+1 : p])
			if isDebug {
				debugf("Header: '%s' Value: '%s'\n", data[:i], headerVal)
			}

			// Headers we need for parsing. Make sure we always
			// capture their value
			if bytes.Equal(headerName, nameContentLength) {
				m.contentLength, _ = parseInt(headerVal)
				m.hasContentLength = true
			} else if bytes.Equal(headerName, nameContentType) {
				m.contentType = headerVal
			} else if bytes.Equal(headerName, nameTransferEncoding) {
				encodings := parseCommaSeparatedList(headerVal)
				// 'chunked' can only appear at the end
				if n := len(encodings); n > 0 && encodings[n-1] == transferEncodingChunked {
					m.isChunked = true
					encodings = encodings[:n-1]
				}
				if len(encodings) > 0 {
					// Append at the end of encodings. If a content-encoding
					// header is also present, it was applied by sender before
					// transfer-encoding.
					m.encodings = append(m.encodings, encodings...)
				}
			} else if bytes.Equal(headerName, nameContentEncoding) {
				encodings := parseCommaSeparatedList(headerVal)
				// Append at the beginning of m.encodings, as Content-Encoding
				// is supposed to be applied before Transfer-Encoding.
				m.encodings = append(encodings, m.encodings...)
			} else if bytes.Equal(headerName, nameConnection) {
				m.connection = headerVal
			} else if len(config.realIPHeader) > 0 && bytes.Equal(headerName, []byte(config.realIPHeader)) {
				if ips := bytes.SplitN(headerVal, []byte{','}, 2); len(ips) > 0 {
					m.realIP = trim(ips[0])
				}
			} else if bytes.Equal(headerName, nameHost) {
				m.host = headerVal
			} else if bytes.Equal(headerName, nameReferer) {
				m.referer = headerVal
			} else if bytes.Equal(headerName, nameUserAgent) {
				m.userAgent = headerVal
			}

			if config.sendHeaders {
				if !config.sendAllHeaders {
					_, exists := config.headersWhitelist[string(headerName)]
					if !exists {
						return true, true, p + 2
					}
				}
				if val, ok := m.headers[string(headerName)]; ok {
					composed := make([]byte, len(val)+len(headerVal)+2)
					off := copy(composed, val)
					copy(composed[off:], []byte(", "))
					copy(composed[off+2:], headerVal)

					m.headers[string(headerName)] = composed
				} else {
					m.headers[string(headerName)] = headerVal
				}
			}

			return true, true, p + 2
		}
	}

	return true, false, len(data)
}

func parseCommaSeparatedList(s common.NetString) (list []string) {
	values := bytes.Split(s, []byte(","))
	list = make([]string, len(values))
	for idx := range values {
		list[idx] = string(bytes.ToLower(bytes.Trim(values[idx], " ")))
	}
	return list
}

func (*parser) parseBody(s *stream, m *message) (ok, complete bool) {
	nbytes := len(s.data)
	if !m.hasContentLength && (bytes.Equal(m.connection, constClose) ||
		(isVersion(m.version, 1, 0) && !bytes.Equal(m.connection, constKeepAlive))) {

		m.size += uint64(nbytes)
		s.bodyReceived += nbytes
		m.contentLength += nbytes

		// HTTP/1.0 no content length. Add until the end of the connection
		if isDebug {
			debugf("http conn close, received %d", len(s.data))
		}
		if m.saveBody {
			m.body = append(m.body, s.data...)
		}
		s.data = nil
		return true, false
	} else if nbytes >= m.contentLength-s.bodyReceived {
		wanted := m.contentLength - s.bodyReceived
		if m.saveBody {
			m.body = append(m.body, s.data[:wanted]...)
		}
		s.bodyReceived = m.contentLength
		m.size += uint64(wanted)
		s.data = s.data[wanted:]
		return true, true
	} else {
		if m.saveBody {
			m.body = append(m.body, s.data...)
		}
		s.data = nil
		s.bodyReceived += nbytes
		m.size += uint64(nbytes)
		if isDebug {
			debugf("bodyReceived: %d", s.bodyReceived)
		}
		return true, false
	}
}

// eatBody acts as if size bytes were received, without having access to
// those bytes.
func (*parser) eatBody(s *stream, m *message, size int) (ok, complete bool) {
	if isDebug {
		debugf("eatBody body")
	}
	if !m.hasContentLength && (bytes.Equal(m.connection, constClose) ||
		(isVersion(m.version, 1, 0) && !bytes.Equal(m.connection, constKeepAlive))) {

		// HTTP/1.0 no content length. Add until the end of the connection
		if isDebug {
			debugf("http conn close, received %d", size)
		}
		m.size += uint64(size)
		s.bodyReceived += size
		m.contentLength += size
		return true, false
	} else if size >= m.contentLength-s.bodyReceived {
		wanted := m.contentLength - s.bodyReceived
		s.bodyReceived += wanted
		m.size = uint64(len(m.rawHeaders) + m.contentLength)
		return true, true
	} else {
		s.bodyReceived += size
		m.size += uint64(size)
		if isDebug {
			debugf("bodyReceived: %d", s.bodyReceived)
		}
		return true, false
	}
}

func (*parser) parseBodyChunkedStart(s *stream, m *message) (cont, ok, complete bool) {
	// read hexa length
	i := bytes.Index(s.data, constCRLF)
	if i == -1 {
		return false, true, false
	}
	line := string(s.data[:i])
	chunkLength, err := strconv.ParseInt(line, 16, 32)
	if err != nil {
		logp.Warn("Failed to understand chunked body start line")
		return false, false, false
	}
	m.chunkedLength = int(chunkLength)

	s.data = s.data[i+2:] //+ \r\n
	m.size += uint64(i + 2)

	if m.chunkedLength == 0 {
		if len(s.data) < 2 {
			s.parseState = stateBodyChunkedWaitFinalCRLF
			return false, true, false
		}
		m.size += 2
		if s.data[0] != '\r' || s.data[1] != '\n' {
			logp.Warn("Expected CRLF sequence at end of message")
			return false, false, false
		}
		s.data = s.data[2:]
		return false, true, true
	}
	s.bodyReceived = 0
	s.parseState = stateBodyChunked

	return true, true, false
}

func (*parser) parseBodyChunked(s *stream, m *message) (cont, ok, complete bool) {
	wanted := m.chunkedLength - s.bodyReceived
	if len(s.data) >= wanted+2 /*\r\n*/ {
		// Received more data than expected
		if m.saveBody {
			m.body = append(m.body, s.data[:wanted]...)
		}
		m.size += uint64(wanted + 2)
		s.data = s.data[wanted+2:]
		m.contentLength += m.chunkedLength
		s.parseState = stateBodyChunkedStart
		return true, true, false
	}

	if len(s.data) >= wanted {
		// we need need to wait for the +2, else we can crash on next call
		return false, true, false
	}

	// Received less data than expected
	if m.saveBody {
		m.body = append(m.body, s.data...)
	}
	s.bodyReceived += len(s.data)
	m.size += uint64(len(s.data))
	s.data = nil
	return false, true, false
}

func (*parser) parseBodyChunkedWaitFinalCRLF(s *stream, m *message) (ok, complete bool) {
	if len(s.data) < 2 {
		return true, false
	}

	m.size += 2
	if s.data[0] != '\r' || s.data[1] != '\n' {
		logp.Warn("Expected CRLF sequence at end of message")
		return false, false
	}

	s.data = s.data[2:]
	return true, true
}

func (parser *parser) shouldIncludeInBody(contenttype []byte, capturedContentTypes []string) bool {
	for _, include := range capturedContentTypes {
		if bytes.Contains(contenttype, []byte(include)) {
			if isDebug {
				debugf("Should Include Body = true Content-Type %s include_body %s",
					contenttype, include)
			}
			return true
		}
	}
	if isDebug {
		debugf("Should Include Body = false Content-Type %s", contenttype)
	}
	return false
}

func (m *message) headersReceived() bool {
	return m.headerOffset > 0
}

func (m *message) getEndpoints() (src *common.Endpoint, dst *common.Endpoint) {
	source, destination := common.MakeEndpointPair(m.tcpTuple.BaseTuple, m.cmdlineTuple)
	src, dst = &source, &destination
	if m.direction == tcp.TCPDirectionReverse {
		src, dst = dst, src
	}
	return src, dst
}

func isVersion(v version, major, minor uint8) bool {
	return v.major == major && v.minor == minor
}

func trim(buf []byte) []byte {
	return trimLeft(trimRight(buf))
}

func trimLeft(buf []byte) []byte {
	for i, b := range buf {
		if b != ' ' && b != '\t' {
			return buf[i:]
		}
	}
	return nil
}

func trimRight(buf []byte) []byte {
	for i := len(buf) - 1; i > 0; i-- {
		b := buf[i]
		if b != ' ' && b != '\t' {
			return buf[:i+1]
		}
	}
	return nil
}

func parseInt(line []byte) (int, error) {
	buf := streambuf.NewFixed(line)
	i, err := buf.IntASCII(false)
	return int(i), err
	// TODO: is it an error if 'buf.Len() != 0 {}' ?
}

func toLower(buf, in []byte) []byte {
	if len(in) > len(buf) {
		goto unbufferedToLower
	}

	for i, b := range in {
		if b > 127 {
			goto unbufferedToLower
		}

		if 'A' <= b && b <= 'Z' {
			b = b - 'A' + 'a'
		}
		buf[i] = b
	}
	return buf[:len(in)]

unbufferedToLower:
	return bytes.ToLower(in)
}
