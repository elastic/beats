package http

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
)

// Http Message
type message struct {
	ts               time.Time
	hasContentLength bool
	headerOffset     int
	version          version
	connection       common.NetString
	chunkedLength    int
	chunkedBody      []byte

	isRequest    bool
	tcpTuple     common.TCPTuple
	cmdlineTuple *common.CmdlineTuple
	direction    uint8

	//Request Info
	requestURI   common.NetString
	method       common.NetString
	statusCode   uint16
	statusPhrase common.NetString
	realIP       common.NetString

	// Http Headers
	contentLength    int
	contentType      common.NetString
	transferEncoding common.NetString
	headers          map[string]common.NetString
	size             uint64

	//Raw Data
	raw []byte

	notes []string

	//Offsets
	start      int
	end        int
	bodyOffset int

	next *message
}

type version struct {
	major uint8
	minor uint8
}

type parser struct {
	config *parserConfig
}

type parserConfig struct {
	realIPHeader     string
	sendHeaders      bool
	sendAllHeaders   bool
	headersWhitelist map[string]bool
}

var (
	transferEncodingChunked = []byte("chunked")

	constCRLF = []byte("\r\n")

	constClose     = []byte("close")
	constKeepAlive = []byte("keep-alive")

	nameContentLength    = []byte("content-length")
	nameContentType      = []byte("content-type")
	nameTransferEncoding = []byte("transfer-encoding")
	nameConnection       = []byte("connection")
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
	m.start = s.parseOffset
	i := bytes.Index(s.data[s.parseOffset:], []byte("\r\n"))
	if i == -1 {
		return false, true, false
	}

	// Very basic tests on the first line. Just to check that
	// we have what looks as an HTTP message
	var version []byte
	var err error
	fline := s.data[s.parseOffset:i]
	if len(fline) < 8 {
		if isDebug {
			debugf("First line too small")
		}
		return false, false, false
	}
	if bytes.Equal(fline[0:5], []byte("HTTP/")) {
		//RESPONSE
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
		slices := bytes.Fields(fline)
		if len(slices) != 3 {
			if isDebug {
				debugf("Couldn't understand HTTP request: %s", fline)
			}
			return false, false, false
		}

		m.method = common.NetString(slices[0])
		m.requestURI = common.NetString(slices[1])

		if bytes.Equal(slices[2][:5], []byte("HTTP/")) {
			m.isRequest = true
			version = slices[2][5:]
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

	p := bytes.IndexByte(s, ' ')
	if p == -1 {
		return 0, nil, errors.New("Not able to identify status code")
	}

	code, _ := parseInt(s[0:p])

	p = bytes.LastIndexByte(s, ' ')
	if p == -1 {
		return uint16(code), nil, errors.New("Not able to identify status code")
	}
	phrase := s[p+1:]
	return uint16(code), phrase, nil
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
		s.parseOffset += 2
		m.bodyOffset = s.parseOffset

		if !m.isRequest && ((100 <= m.statusCode && m.statusCode < 200) || m.statusCode == 204 || m.statusCode == 304) {
			//response with a 1xx, 204 , or 304 status  code is always terminated
			// by the first empty line after the  header fields
			if isDebug {
				debugf("Terminate response, status code %d", m.statusCode)
			}
			m.end = s.parseOffset
			m.size = uint64(m.end - m.start)
			return false, true, true
		}

		if bytes.Equal(m.transferEncoding, transferEncodingChunked) {
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
			m.end = s.parseOffset
			m.size = uint64(m.end - m.start)
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
				m.transferEncoding = common.NetString(headerVal)
			} else if bytes.Equal(headerName, nameConnection) {
				m.connection = headerVal
			}
			if len(config.realIPHeader) > 0 && bytes.Equal(headerName, []byte(config.realIPHeader)) {
				if ips := bytes.SplitN(headerVal, []byte{','}, 2); len(ips) > 0 {
					m.realIP = trim(ips[0])
				}
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
					off = copy(composed[off:], []byte(", "))
					copy(composed[off:], headerVal)

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

func (*parser) parseBody(s *stream, m *message) (ok, complete bool) {
	if isDebug {
		debugf("parseBody body: %d", s.parseOffset)
	}
	if !m.hasContentLength && (bytes.Equal(m.connection, constClose) ||
		(isVersion(m.version, 1, 0) && !bytes.Equal(m.connection, constKeepAlive))) {

		// HTTP/1.0 no content length. Add until the end of the connection
		if isDebug {
			debugf("http conn close, received %d", len(s.data)-s.parseOffset)
		}
		s.bodyReceived += (len(s.data) - s.parseOffset)
		m.contentLength += (len(s.data) - s.parseOffset)
		s.parseOffset = len(s.data)
		return true, false
	} else if len(s.data[s.parseOffset:]) >= m.contentLength-s.bodyReceived {
		s.parseOffset += (m.contentLength - s.bodyReceived)
		m.end = s.parseOffset
		m.size = uint64(m.end - m.start)
		return true, true
	} else {
		s.bodyReceived += (len(s.data) - s.parseOffset)
		s.parseOffset = len(s.data)
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
		debugf("eatBody body: %d", s.parseOffset)
	}
	if !m.hasContentLength && (bytes.Equal(m.connection, constClose) ||
		(isVersion(m.version, 1, 0) && !bytes.Equal(m.connection, constKeepAlive))) {

		// HTTP/1.0 no content length. Add until the end of the connection
		if isDebug {
			debugf("http conn close, received %d", size)
		}
		s.bodyReceived += size
		m.contentLength += size
		return true, false
	} else if size >= m.contentLength-s.bodyReceived {
		s.bodyReceived += (m.contentLength - s.bodyReceived)
		m.end = s.parseOffset
		m.size = uint64(m.bodyOffset-m.start) + uint64(m.contentLength)
		return true, true
	} else {
		s.bodyReceived += size
		if isDebug {
			debugf("bodyReceived: %d", s.bodyReceived)
		}
		return true, false
	}
}

func (*parser) parseBodyChunkedStart(s *stream, m *message) (cont, ok, complete bool) {
	// read hexa length
	i := bytes.Index(s.data[s.parseOffset:], constCRLF)
	if i == -1 {
		return false, true, false
	}
	line := string(s.data[s.parseOffset : s.parseOffset+i])
	_, err := fmt.Sscanf(line, "%x", &m.chunkedLength)
	if err != nil {
		logp.Warn("Failed to understand chunked body start line")
		return false, false, false
	}

	s.parseOffset += i + 2 //+ \r\n
	if m.chunkedLength == 0 {
		if len(s.data[s.parseOffset:]) < 2 {
			s.parseState = stateBodyChunkedWaitFinalCRLF
			return false, true, false
		}
		if s.data[s.parseOffset] != '\r' || s.data[s.parseOffset+1] != '\n' {
			logp.Warn("Expected CRLF sequence at end of message")
			return false, false, false
		}
		s.parseOffset += 2 // skip final CRLF

		m.end = s.parseOffset
		m.size = uint64(m.end - m.start)
		return false, true, true
	}
	s.bodyReceived = 0
	s.parseState = stateBodyChunked

	return true, true, false
}

func (*parser) parseBodyChunked(s *stream, m *message) (cont, ok, complete bool) {

	if len(s.data[s.parseOffset:]) >= m.chunkedLength-s.bodyReceived+2 /*\r\n*/ {
		// Received more data than expected
		m.chunkedBody = append(m.chunkedBody, s.data[s.parseOffset:s.parseOffset+m.chunkedLength-s.bodyReceived]...)
		s.parseOffset += (m.chunkedLength - s.bodyReceived + 2 /*\r\n*/)
		m.contentLength += m.chunkedLength
		s.parseState = stateBodyChunkedStart
		return true, true, false
	}

	if len(s.data[s.parseOffset:]) >= m.chunkedLength-s.bodyReceived {
		// we need need to wait for the +2, else we can crash on next call
		return false, true, false
	}

	// Received less data than expected
	m.chunkedBody = append(m.chunkedBody, s.data[s.parseOffset:]...)
	s.bodyReceived += (len(s.data) - s.parseOffset)
	s.parseOffset = len(s.data)
	return false, true, false
}

func (*parser) parseBodyChunkedWaitFinalCRLF(s *stream, m *message) (ok, complete bool) {
	if len(s.data[s.parseOffset:]) < 2 {
		return true, false
	}

	if s.data[s.parseOffset] != '\r' || s.data[s.parseOffset+1] != '\n' {
		logp.Warn("Expected CRLF sequence at end of message")
		return false, false
	}

	s.parseOffset += 2 // skip final CRLF
	m.end = s.parseOffset
	m.size = uint64(m.end - m.start)
	return true, true
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
