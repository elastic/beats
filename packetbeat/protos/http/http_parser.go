package http

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Http Message
type message struct {
	Ts               time.Time
	hasContentLength bool
	headerOffset     int
	bodyOffset       int
	version          version
	connection       string
	chunkedLength    int
	chunkedBody      []byte

	IsRequest    bool
	TCPTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
	Direction    uint8
	//Request Info
	FirstLine    string
	RequestURI   string
	Method       string
	StatusCode   uint16
	StatusPhrase string
	RealIP       string
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
	RealIPHeader     string
	SendHeaders      bool
	SendAllHeaders   bool
	HeadersWhitelist map[string]bool
}

func newParser(config *parserConfig) *parser {
	return &parser{config: config}
}

func (parser *parser) parse(s *stream) (bool, bool) {
	m := s.message

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
		debugf("First line too small")
		return false, false, false
	}
	if bytes.Equal(fline[0:5], []byte("HTTP/")) {
		//RESPONSE
		m.IsRequest = false
		version = fline[5:8]
		m.StatusCode, m.StatusPhrase, err = parseResponseStatus(fline[9:])
		if err != nil {
			logp.Warn("Failed to understand HTTP response status: %s", fline[9:])
			return false, false, false
		}
		debugf("HTTP status_code=%d, status_phrase=%s", m.StatusCode, m.StatusPhrase)

	} else {
		// REQUEST
		slices := bytes.Fields(fline)
		if len(slices) != 3 {
			debugf("Couldn't understand HTTP request: %s", fline)
			return false, false, false
		}

		m.Method = string(slices[0])
		m.RequestURI = string(slices[1])

		if bytes.Equal(slices[2][:5], []byte("HTTP/")) {
			m.IsRequest = true
			version = slices[2][5:]
			m.FirstLine = string(fline)
		} else {
			debugf("Couldn't understand HTTP version: %s", fline)
			return false, false, false
		}
		debugf("HTTP Method=%s, RequestUri=%s", m.Method, m.RequestURI)
	}

	m.version.major, m.version.minor, err = parseVersion(version)
	if err != nil {
		debugf("Failed to understand HTTP version: %v", version)
		m.version.major = 1
		m.version.minor = 0
	}
	debugf("HTTP version %d.%d", m.version.major, m.version.minor)

	// ok so far
	s.parseOffset = i + 2
	m.headerOffset = s.parseOffset
	s.parseState = stateHeaders

	return true, true, true
}

func parseResponseStatus(s []byte) (uint16, string, error) {
	debugf("parseResponseStatus: %s", s)

	p := bytes.Index(s, []byte(" "))
	if p == -1 {
		return 0, "", errors.New("Not beeing able to identify status code")
	}

	code, _ := strconv.Atoi(string(s[0:p]))

	p = bytes.LastIndex(s, []byte(" "))
	if p == -1 {
		return uint16(code), "", errors.New("Not beeing able to identify status code")
	}
	phrase := s[p+1:]
	return uint16(code), string(phrase), nil
}

func parseVersion(s []byte) (uint8, uint8, error) {
	if len(s) < 3 {
		return 0, 0, errors.New("Invalid version")
	}

	major, _ := strconv.Atoi(string(s[0]))
	minor, _ := strconv.Atoi(string(s[2]))

	return uint8(major), uint8(minor), nil
}

func (parser *parser) parseHeaders(s *stream, m *message) (cont, ok, complete bool) {
	if len(s.data)-s.parseOffset >= 2 &&
		bytes.Equal(s.data[s.parseOffset:s.parseOffset+2], []byte("\r\n")) {
		// EOH
		s.parseOffset += 2
		m.bodyOffset = s.parseOffset

		if !m.IsRequest && ((100 <= m.StatusCode && m.StatusCode < 200) || m.StatusCode == 204 || m.StatusCode == 304) {
			//response with a 1xx, 204 , or 304 status  code is always terminated
			// by the first empty line after the  header fields
			debugf("Terminate response, status code %d", m.StatusCode)
			m.end = s.parseOffset
			m.Size = uint64(m.end - m.start)
			return false, true, true
		}

		if m.TransferEncoding == "chunked" {
			// support for HTTP/1.1 Chunked transfer
			// Transfer-Encoding overrides the Content-Length
			debugf("Read chunked body")
			s.parseState = stateBodyChunkedStart
			return true, true, true
		}

		if m.ContentLength == 0 && (m.IsRequest || m.hasContentLength) {
			debugf("Empty content length, ignore body")
			// Ignore body for request that contains a message body but not a Content-Length
			m.end = s.parseOffset
			m.Size = uint64(m.end - m.start)
			return false, true, true
		}

		debugf("Read body")
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
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	i := bytes.Index(data, []byte(":"))
	if i == -1 {
		// Expected \":\" in headers. Assuming incomplete"
		return true, false, 0
	}

	config := parser.config

	detailedf("Data: %s", data)
	detailedf("Header: %s", data[:i])

	// skip folding line
	for p := i + 1; p < len(data); {
		q := bytes.Index(data[p:], []byte("\r\n"))
		if q == -1 {
			// Assuming incomplete
			return true, false, 0
		}
		p += q
		detailedf("HV: %s\n", data[i+1:p])
		if len(data) > p && (data[p+1] == ' ' || data[p+1] == '\t') {
			p = p + 2
		} else {
			headerName := strings.ToLower(string(data[:i]))
			headerVal := string(bytes.Trim(data[i+1:p], " \t"))
			debugf("Header: '%s' Value: '%s'\n", headerName, headerVal)

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
			if len(config.RealIPHeader) > 0 && headerName == config.RealIPHeader {
				m.RealIP = headerVal
			}

			if config.SendHeaders {
				if !config.SendAllHeaders {
					_, exists := config.HeadersWhitelist[headerName]
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

func (*parser) parseBody(s *stream, m *message) (ok, complete bool) {
	debugf("eat body: %d", s.parseOffset)
	if !m.hasContentLength && (m.connection == "close" ||
		(isVersion(m.version, 1, 0) && m.connection != "keep-alive")) {

		// HTTP/1.0 no content length. Add until the end of the connection
		debugf("close connection, %d", len(s.data)-s.parseOffset)
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
		debugf("bodyReceived: %d", s.bodyReceived)
		return true, false
	}
}

func (*parser) parseBodyChunkedStart(s *stream, m *message) (cont, ok, complete bool) {
	// read hexa length
	i := bytes.Index(s.data[s.parseOffset:], []byte("\r\n"))
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
		m.Size = uint64(m.end - m.start)
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
		m.ContentLength += m.chunkedLength
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
	m.Size = uint64(m.end - m.start)
	return true, true
}

func isVersion(v version, major, minor uint8) bool {
	return v.major == major && v.minor == minor
}
