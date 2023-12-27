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

package sip

import (
	"bytes"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/streambuf"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

// sip Message
type message struct {
	ts               time.Time
	hasContentLength bool
	headerOffset     int

	isRequest bool

	// Info
	requestURI   common.NetString
	method       common.NetString
	statusCode   uint16
	statusPhrase common.NetString
	version      version

	// Headers
	contentLength int
	contentType   common.NetString
	userAgent     common.NetString
	to            common.NetString
	from          common.NetString
	cseq          common.NetString
	callID        common.NetString
	maxForwards   int
	viaDedup      map[string]struct{}
	via           []common.NetString
	allow         []string
	supported     []string

	headers map[string][]common.NetString
	size    uint64

	firstLine  []byte
	rawHeaders []byte
	body       []byte
	rawData    []byte
}

type version struct {
	major uint8
	minor uint8
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

type parserState uint8

const (
	stateStart parserState = iota
	stateHeaders
	stateBody
)

type parsingInfo struct {
	pkt *protos.Packet

	data []byte

	parseOffset  int
	state        parserState
	bodyReceived int

	message *message
}

func (pi *parsingInfo) prepareForNewMessage() {
	pi.state = stateStart
	pi.parseOffset = 0
	pi.bodyReceived = 0
	pi.message = nil
}

func newParsingInfo(pkt *protos.Packet, tuple common.BaseTuple) *parsingInfo {
	return &parsingInfo{
		pkt:  pkt,
		data: pkt.Payload,
	}
}

var (
	constCRLF         = []byte("\r\n")
	constSIPVersion   = []byte("SIP/")
	nameContentLength = []byte("content-length")
	nameContentType   = []byte("content-type")
	nameUserAgent     = []byte("user-agent")
	nameTo            = []byte("to")
	nameFrom          = []byte("from")
	nameCseq          = []byte("cseq")
	nameCallID        = []byte("call-id")
	nameMaxForwards   = []byte("max-forwards")
	nameAllow         = []byte("allow")
	nameSupported     = []byte("supported")
	nameVia           = []byte("via")
)

func parse(pi *parsingInfo) (ok, complete bool) {
	m := pi.message
	for pi.parseOffset < len(pi.data) {
		switch pi.state {
		case stateStart:
			if ok, cont, complete := parseSIPLine(pi, m); !cont {
				return ok, complete
			}
		case stateHeaders:
			if ok, cont, complete := parseHeaders(pi, m); !cont {
				return ok, complete
			}
		case stateBody:
			return parseBody(pi, m)
		}
	}
	return true, false
}

func parseSIPLine(pi *parsingInfo, m *message) (ok, cont, complete bool) {
	// ignore any CRLF appearing before the start-line (RFC3261 7.5)
	pi.data = bytes.TrimLeft(pi.data[pi.parseOffset:], string(constCRLF))

	i := bytes.Index(pi.data[pi.parseOffset:], constCRLF)
	if i == -1 {
		return true, false, false
	}

	// Very basic tests on the first line. Just to check that
	// we have what looks as a SIP message
	var (
		version []byte
		err     error
	)

	const minStatusLineLength = len("SIP/2.0 XXX OK")
	fline := pi.data[pi.parseOffset:i]
	if len(fline) < minStatusLineLength {
		if isDebug {
			debugf("First line too small")
		}
		return false, false, false
	}

	m.firstLine = fline

	if bytes.Equal(fline[0:4], constSIPVersion) {
		// RESPONSE
		m.isRequest = false
		version = fline[4:7]
		m.statusCode, m.statusPhrase, err = parseResponseStatus(fline[8:])
		if err != nil {
			if isDebug {
				debugf("Failed to understand SIP response status: %s", fline[8:])
			}
			return false, false, false
		}

		if isDebug {
			debugf("SIP status_code=%d, status_phrase=%s", m.statusCode, m.statusPhrase)
		}
	} else {
		// REQUEST
		afterMethodIdx := bytes.IndexFunc(fline, unicode.IsSpace)
		afterRequestURIIdx := bytes.LastIndexFunc(fline, unicode.IsSpace)

		// Make sure we have the VERB + URI + SIP_VERSION
		if afterMethodIdx == -1 || afterRequestURIIdx == -1 || afterMethodIdx == afterRequestURIIdx {
			if isDebug {
				debugf("Couldn't understand SIP request: %s", fline)
			}
			return false, false, false
		}

		m.method = common.NetString(fline[:afterMethodIdx])
		m.requestURI = common.NetString(fline[afterMethodIdx+1 : afterRequestURIIdx])

		versionIdx := afterRequestURIIdx + len(constSIPVersion) + 1
		if len(fline) > versionIdx && bytes.Equal(fline[afterRequestURIIdx+1:versionIdx], constSIPVersion) {
			m.isRequest = true
			version = fline[versionIdx:]
		} else {
			if isDebug {
				debugf("Couldn't understand SIP version: %s", fline)
			}
			return false, false, false
		}
	}

	m.version.major, m.version.minor, err = parseVersion(version)
	if err != nil {
		if isDebug {
			debugf(err.Error(), version)
		}
		return false, false, false
	}
	if isDebug {
		debugf("SIP version %d.%d", m.version.major, m.version.minor)
	}

	// ok so far
	pi.parseOffset = i + 2
	m.headerOffset = pi.parseOffset
	pi.state = stateHeaders

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

	return major, minor, nil
}

func parseHeaders(pi *parsingInfo, m *message) (ok, cont, complete bool) {
	// check if it isn't headers end yet with /r/n/r/n
	if len(pi.data)-pi.parseOffset < 2 || !bytes.Equal(pi.data[pi.parseOffset:pi.parseOffset+2], constCRLF) {
		ok, hcomplete, offset := parseHeader(pi, m)
		if !ok {
			return false, false, false
		}

		if !hcomplete {
			return true, false, false
		}

		pi.parseOffset += offset

		return true, true, true
	}

	m.size = uint64(pi.parseOffset + 2)
	m.rawHeaders = pi.data[:m.size]
	pi.data = pi.data[m.size:]
	pi.parseOffset = 0

	if m.contentLength == 0 && (m.isRequest || m.hasContentLength) {
		if isDebug {
			debugf("Empty content length, ignore body")
		}
		return true, false, true
	}

	if isDebug {
		debugf("Read body")
	}

	pi.state = stateBody

	return true, true, true
}

func parseHeader(pi *parsingInfo, m *message) (ok, complete bool, offset int) {
	if m.headers == nil {
		m.headers = make(map[string][]common.NetString)
	}

	data := pi.data[pi.parseOffset:]

	i := bytes.Index(data, []byte(":"))
	if i == -1 {
		// Expected \":\" in headers. Assuming incomplete
		return true, false, 0
	}

	// enabled if required. Allocs for parameters slow down parser big times
	if isDetailed {
		detailedf("Data: %s", data)
		detailedf("Header: %s", data[:i])
	}

	// skip folding line
	for p := i + 1; p < len(data); {
		q := bytes.Index(data[p:], constCRLF)
		if q == -1 {
			// assuming incomplete
			return true, false, 0
		}

		p += q
		if len(data) > p && (data[p+1] == ' ' || data[p+1] == '\t') {
			p = p + 2
			continue
		}

		headerName := getExpandedHeaderName(bytes.ToLower(data[:i]))
		headerVal := bytes.TrimSpace(data[i+1 : p])
		if isDebug {
			debugf("Header: '%s' Value: '%s'\n", data[:i], headerVal)
		}

		// Headers we need for parsing. Make sure we always
		// capture their value
		switch {
		case bytes.Equal(headerName, nameMaxForwards):
			m.maxForwards, _ = parseInt(headerVal)
		case bytes.Equal(headerName, nameContentLength):
			m.contentLength, _ = parseInt(headerVal)
			m.hasContentLength = true
		case bytes.Equal(headerName, nameContentType):
			m.contentType = headerVal
		case bytes.Equal(headerName, nameUserAgent):
			m.userAgent = headerVal
		case bytes.Equal(headerName, nameTo):
			m.to = headerVal
		case bytes.Equal(headerName, nameFrom):
			m.from = headerVal
		case bytes.Equal(headerName, nameCseq):
			m.cseq = headerVal
		case bytes.Equal(headerName, nameCallID):
			m.callID = headerVal
		case bytes.Equal(headerName, nameAllow):
			m.allow = parseCommaSeparatedList(headerVal)
		case bytes.Equal(headerName, nameSupported):
			m.supported = parseCommaSeparatedList(headerVal)
		case bytes.Equal(headerName, nameVia):
			if m.viaDedup == nil {
				m.viaDedup = map[string]struct{}{}
			}
			if _, found := m.viaDedup[string(headerVal)]; !found {
				m.via = append(m.via, headerVal)
				m.viaDedup[string(headerVal)] = struct{}{}
			}
		}

		m.headers[string(headerName)] = append(
			m.headers[string(headerName)],
			headerVal,
		)

		return true, true, p + 2
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

func parseBody(pi *parsingInfo, m *message) (ok, complete bool) {
	nbytes := len(pi.data)
	if nbytes >= m.contentLength-pi.bodyReceived {
		wanted := m.contentLength - pi.bodyReceived
		m.body = append(m.body, pi.data[:wanted]...)
		pi.bodyReceived = m.contentLength
		m.size += uint64(wanted)
		pi.data = pi.data[wanted:]
		return true, true
	}
	m.body = append(m.body, pi.data...)
	pi.data = nil
	pi.bodyReceived += nbytes
	m.size += uint64(nbytes)
	if isDebug {
		debugf("bodyReceived: %d", pi.bodyReceived)
	}
	return true, false
}

func parseInt(line []byte) (int, error) {
	buf := streambuf.NewFixed(line)
	i, err := buf.IntASCII(false)
	return int(i), err
	// TODO: is it an error if 'buf.Len() != 0 {}' ?
}

func getExpandedHeaderName(n []byte) []byte {
	if len(n) > 1 {
		return n
	}
	switch string(n) {
	// referfenced by https://www.iana.org/assignments/sip-parameters/sip-parameters.xhtml
	case "a":
		return []byte("accept-contact") //[RFC3841]
	case "b":
		return []byte("referred-by") //[RFC3892]
	case "c":
		return []byte("content-type") //[RFC3261]
	case "d":
		return []byte("request-disposition") //[RFC3841]
	case "e":
		return []byte("content-encoding") //[RFC3261]
	case "f":
		return []byte("from") //[RFC3261]
	case "i":
		return []byte("call-id") //[RFC3261]
	case "j":
		return []byte("reject-contact") //[RFC3841]
	case "k":
		return []byte("supported") //[RFC3261]
	case "l":
		return []byte("content-length") //[RFC3261]
	case "m":
		return []byte("contact") //[RFC3261]
	case "o":
		return []byte("event") //[RFC666)5] [RFC6446]
	case "r":
		return []byte("refer-to") //[RFC3515]
	case "s":
		return []byte("subject") //[RFC3261]
	case "t":
		return []byte("to") //[RFC3261]
	case "u":
		return []byte("allow-events") //[RFC6665]
	case "v":
		return []byte("via") //[RFC326)1] [RFC7118]
	case "x":
		return []byte("session-expires") //[RFC4028]
	case "y":
		return []byte("identity") //[RFC8224]
	}
	return n
}
