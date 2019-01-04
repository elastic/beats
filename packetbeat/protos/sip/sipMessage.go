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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

/**
 ******************************************************************
 * sipMessage
 *******************************************************************
 **/

// SipMessage contains a single SIP message.
type sipMessage struct {
	ts           time.Time          // Time when the message was received.
	tuple        common.IPPortTuple // Source and destination addresses of packet.
	cmdlineTuple *common.ProcessTuple
	transport    transport

	// SIP FirstLines
	isRequest    bool
	method       common.NetString
	requestURI   common.NetString
	statusCode   uint16
	statusPhrase common.NetString

	// SIP Headers
	from          common.NetString
	to            common.NetString
	cseq          common.NetString
	callid        common.NetString
	headers       *map[string][]common.NetString
	contentlength int

	// SIP Bodies
	body map[string]*map[string][]common.NetString

	// Raw Data
	raw []byte

	// Additional Information
	notes []common.NetString

	// Offsets
	hdrStart int
	hdrLen   int
	bdyStart int

	// flags
	isIncompletedHdrMsg bool
	isIncompletedBdyMsg bool
}

func (msg sipMessage) String() string {
	outputs := ""
	outputs += fmt.Sprintf("%s:Src:%s:%d -> Dst:%s:%d ,", msg.ts,
		msg.tuple.SrcIP,
		msg.tuple.SrcPort,
		msg.tuple.DstIP,
		msg.tuple.DstPort)
	if msg.isRequest {
		outputs += "Request: ("
		outputs += string(msg.method)
		outputs += ", "
		outputs += string(msg.requestURI)
		outputs += "), "
	} else {
		outputs += "Response: ("
		outputs += fmt.Sprintf("%03d", msg.statusCode)
		outputs += ", "
		outputs += string(msg.statusPhrase)
		outputs += "), "
	}
	outputs += " From   : " + string(msg.from) + ", "
	outputs += " To     : " + string(msg.to) + ", "
	outputs += " CSeq   : " + string(msg.cseq) + ", "
	outputs += " Call-ID: " + string(msg.callid) + ", "

	if msg.headers != nil {
		outputs += " Headers: ["
		for header, array := range *(msg.headers) {
			for idx, line := range array {
				outputs += fmt.Sprintf(" { %20s[%3d] : %s} ", header, idx, line)
			}
		}
	}
	if msg.body != nil {
		outputs += ", body: "
		for body, mapsP := range msg.body {
			outputs += fmt.Sprintf("{ %s : ", body)
			if body == "application/sdp" {
				for key, lines := range *mapsP {
					for idx, line := range lines {
						outputs += fmt.Sprintf("  { %5s[%3d] : %s } ", key, idx, line)
					}
				}
			}
			outputs += fmt.Sprintf(" }")
		}
	}
	return outputs
}
func (msg *sipMessage) parseSIPHeader() (err error) {
	msg.hdrStart = -1
	msg.hdrLen = -1
	msg.bdyStart = -1
	msg.contentlength = -1

	// Find SIP header start position and headers-bodies separeted postion
	cutPosS := []int{} // SIP message start point and after CRLF points
	cutPosE := []int{} // before CRLF points

	byteLen := len(msg.raw)
	hdrStart := -1      // SIP message statr point was initialized by -1
	hdrEnd := -1        // SIP header end point(\r\n\r\n) was initialized by -1
	bdyStart := byteLen // SIP bodies start point (after \r\n\r\n) was initialized by packet length

	for i, ch := range msg.raw {
		// ignore any CRLF appearing before the start-line (RFC3261 7.5)
		if hdrStart == -1 {
			if ch == byte('\n') || ch == byte('\r') {
				continue
			} else {
				cutPosS = append(cutPosS, i)
				hdrStart = i
			}
		}

		// getting all CRLF points
		if i+1 < byteLen &&
			msg.raw[i+0] == byte('\r') && msg.raw[i+1] == byte('\n') {
			cutPosE = append(cutPosE, i)
			cutPosS = append(cutPosS, i+2)
		}
		// getting header break point
		if i+3 < byteLen &&
			msg.raw[i+0] == byte('\r') && msg.raw[i+1] == byte('\n') &&
			msg.raw[i+2] == byte('\r') && msg.raw[i+3] == byte('\n') {
			hdrEnd = i
			bdyStart = i + 4
			break
		}
	}

	// Set finded point to sipMessage member field
	msg.hdrStart = hdrStart

	// in case hdrStar == -1,
	// it is means that the packet was padded with CRLFs
	// return errors
	if hdrStart < 0 {
		return fmt.Errorf("malformed packet")
	}

	// in case missing header end point
	// it is means that the packet was incomplete as SIP message
	// flag the indicator
	if hdrEnd < 0 {
		msg.isIncompletedHdrMsg = true
		hdrEnd = byteLen
	}

	// calculate header length by header endpoint and startpoint
	msg.hdrLen = hdrEnd - hdrStart
	msg.bdyStart = bdyStart

	// parse SIP header and getting maps
	headers, startLine := msg.parseSIPHeaderToMap(cutPosS, cutPosE)

	// in case start line was malformed, return error
	if len(startLine) != 3 {
		msg.notes = append(msg.notes, common.NetString("start line parse error."))
		return fmt.Errorf("malformed packet")
	}

	// decide request or response
	msg.isRequest = strings.Contains(startLine[2], "SIP/2.0")
	if msg.isRequest {
		msg.method = common.NetString(startLine[0])
		msg.requestURI = common.NetString(startLine[1])
	} else if strings.Contains(startLine[0], "SIP/2.0") { // Response
		parsedStatusCode, err := strconv.ParseInt(startLine[1], 10, 16)
		if err != nil {
			msg.statusCode = uint16(999)
			msg.notes = append(msg.notes, common.NetString(fmt.Sprintf("invalid status-code %s", startLine[1])))
		} else {
			msg.statusCode = uint16(parsedStatusCode)
		}
		msg.statusPhrase = common.NetString(strings.TrimSpace(startLine[2]))
	} else {
		msg.notes = append(msg.notes, common.NetString("malformed packet. this is not sip message."))
		return fmt.Errorf("malformed packet(this is not sip message)")
	}

	// mandatory header fields check
	toArray, existTo := (*headers)["to"]
	fromArray, existFrom := (*headers)["from"]
	cseqArray, existCSeq := (*headers)["cseq"]
	callidArray, existCallID := (*headers)["call-id"]
	maxfrowardsArray, existMaxForwards := (*headers)["max-forwards"]
	viaArray, existVia := (*headers)["via"]

	if existTo {
		msg.to = getLastElementStrArray(toArray)
	} else {
		msg.notes = append(msg.notes, common.NetString("mandatory header [To] does not exist."))
	}

	if existFrom {
		msg.from = getLastElementStrArray(fromArray)
	} else {
		msg.notes = append(msg.notes, common.NetString("mandatory header [From] does not exist."))
	}

	if existCSeq {
		msg.cseq = getLastElementStrArray(cseqArray)
	} else {
		msg.notes = append(msg.notes, common.NetString("mandatory header [CSeq] does not exist."))
	}

	if existCallID {
		msg.callid = getLastElementStrArray(callidArray)
	} else {
		msg.notes = append(msg.notes, common.NetString("mandatory header [Call-ID] does not exist."))
	}

	if !existMaxForwards {
	}
	if !existVia {
		msg.notes = append(msg.notes, common.NetString("mandatory header [Via] does not exist."))
	}

	// headers value update
	msg.headers = headers

	// unused
	_ = maxfrowardsArray
	_ = viaArray

	// Content-Lenght initialized to 0
	msg.contentlength = 0
	contenttypeArray, existContentType := (*headers)["content-type"]
	contentlengthArray, existContentLength := (*headers)["content-length"]
	_ = contenttypeArray

	contentlength := 0

	if existContentType {
		// in case Content-Length was exist
		// getting content-Length with header values
		// in case parseint missed , lenght was reset with 0
		if existContentLength {
			rawCntLen, errCntLen := strconv.ParseInt(string(getLastElementStrArray(contentlengthArray)), 10, 64)
			contentlength = int(rawCntLen)

			// parseint error, 0 reset
			if errCntLen != nil {
				contentlength = 0
			}
		} else {
			contentlength = byteLen - bdyStart
		}
	} else {
		// in case content-type was not founded from packet
		// bodies was ignored (RFC 3261 20.15)
		contentlength = 0
	}

	msg.contentlength = contentlength

	if msg.bdyStart+msg.contentlength > byteLen {
		// in case bodies length was short than content-length
		// flag the indicator
		msg.isIncompletedBdyMsg = true
		msg.contentlength = -1
	}

	return nil
}

/**
 * commaSeparatedString is string split with comma, the string was split with comma,
 * trim the SPs and convert to NetString and thats return as an array.
 *
 * example:
 * commaSeparatedString : ,aaaa,"bbbb,ccc",hoge\,hige,\"aa,aa\",
 * separatedStrings :
 *  [0]:
 *  [1]: aaaa
 *  [2]: "bbbb,cccc"
 *  [3]: hoge\,hige
 *  [4]: \"aa
 *  [5]: aa\"
 *  [6]:
 *
 *  example2:
 *  commaSeparatedString : aaaa,"aaaaa,bbb
 *  separatedStrings :
 *   [0]: aaaa
 *   [1]: "aaaaa,bbb | output immediately during escaped finished
 **/
func (msg *sipMessage) separateCsv(commaSeparatedString string) (separatedStrings *[]common.NetString) {
	separatedStrings = &[]common.NetString{}
	var prevChar rune
	startIdx := 0
	insubcsv := false
	escaped := false
	for idx, curChar := range commaSeparatedString {
		/* MEMO:state of escaped bool
		 *   time|01234567
		 * ------+--------
		 *   char| \\"\\\"
		 * ------+--------
		 * x=!esc| TFTTFTF //
		 * y=c==\| TTFTTTF // result of prevChar==\\
		 * ------+--------
		 *   x&&y|FTFFTFTF // calculation result of escaped bool
		 */
		escaped = (!escaped && prevChar == '\\')
		finalChr := (idx+1 == len(commaSeparatedString))
		isComma := (curChar == ',')

		if curChar == '"' && !escaped {
			insubcsv = !insubcsv
		}

		if finalChr {
			if isComma && !insubcsv {
				subStr := strings.TrimSpace(commaSeparatedString[startIdx:idx])
				*separatedStrings = append(*separatedStrings, common.NetString(subStr))
				*separatedStrings = append(*separatedStrings, common.NetString(""))
			} else {
				subStr := strings.TrimSpace(commaSeparatedString[startIdx : idx+1])
				*separatedStrings = append(*separatedStrings, common.NetString(subStr))
			}
		} else if !insubcsv && (!escaped && isComma) {
			subStr := strings.TrimSpace(commaSeparatedString[startIdx:idx])
			*separatedStrings = append(*separatedStrings, common.NetString(subStr))

			startIdx = idx + 1
		}
		prevChar = curChar
	}

	return separatedStrings
}

func (msg *sipMessage) parseSIPHeaderToMap(cutPosS []int, cutPosE []int) (*map[string][]common.NetString, []string) {
	firstLines := []string{}
	headers := &map[string][]common.NetString{}

	var lastheader string
	for i := 0; i < len(cutPosE); i++ {
		s := cutPosS[i]
		e := cutPosE[i]

		if i == 0 {
			// Request-line or Status-line is set to firstLines
			firstLines = strings.SplitN(string(msg.raw[s:e]), " ", 3)
		} else {
			//  Header fields can be extended over multiple lines by preceding each
			// extra line with at least one SP or horizontal tab (HT).(RFC3261 7.3.1)
			// (A)
			// Subject: I know you're there, pick up the phone and talk to me!
			// (B)
			// Subject: I know you're there,
			//          pick up the phone
			//          and talk to me!
			// (A) and (B) are equivalent
			if msg.raw[s] == byte(' ') || msg.raw[s] == byte('\t') {
				if lastheader != "" {
					lastElement := string(getLastElementStrArray((*headers)[lastheader]))
					// TrimSpace is delete both " " and "\t"
					lastElement += fmt.Sprintf(" %s", strings.TrimSpace(string(msg.raw[s:e])))
					(*headers)[lastheader][len((*headers)[lastheader])-1] = common.NetString(lastElement)
				} else {
					// ignore this line
				}
				continue
			}

			// in case header value was comma separated strings (ex. Hoge: hige, foo)
			// FIXME: Above process
			if lastheader != "" {
				lastHeaderEndIdx := len((*headers)[lastheader]) - 1
				if lastHeaderEndIdx < 0 {
					continue
				} // This case is not exist, maybe...

				lastElement := string(getLastElementStrArray((*headers)[lastheader]))
				separatedStrings := msg.separateCsv(lastElement)
				for idx, element := range *separatedStrings {
					if idx == 0 {
						(*headers)[lastheader][lastHeaderEndIdx] = element
					} else {
						(*headers)[lastheader] = append((*headers)[lastheader], element)
					}
				}
			}

			// in case header line is NOT start [SP] or [HT]
			// this line will be header parameter
			// header parameter shuld be include ':'
			// and split two string ,before first ':' and after first ':'.
			headerKeyVal := strings.SplitN(string(msg.raw[s:e]), ":", 2)
			key := strings.ToLower(strings.TrimSpace(headerKeyVal[0]))
			val := strings.TrimSpace(headerKeyVal[1])

			// in case string was not included the ':', it is not valid data, ignored.
			if val == "" {
				continue
			}

			// Convert compact form to full form
			if len(key) == 1 {
				if val, ok := cmpctFormToFullForm[key[0]]; ok {
					key=val
				}
			}

			// Initialize and add to map, if first find the header name in process
			_, ok := (*headers)[key]
			if !ok {
				(*headers)[key] = []common.NetString{}
			}

			(*headers)[key] = append((*headers)[key], common.NetString(val))
			lastheader = key
		}
		// in case last processed headers line is separated with comma.
		// FIXME: this process is same to "Above process"
		if lastheader != "" {
			lastHeaderEndIdx := len((*headers)[lastheader]) - 1
			if lastHeaderEndIdx < 0 {
				continue
			} // This case is not exist, maybe...

			lastElement := string(getLastElementStrArray((*headers)[lastheader]))
			separatedStrings := msg.separateCsv(lastElement)
			for idx, element := range *separatedStrings {
				if idx == 0 {
					(*headers)[lastheader][lastHeaderEndIdx] = element
				} else {
					(*headers)[lastheader] = append((*headers)[lastheader], element)
				}
			}
		}
	}
	return headers, firstLines
}

// TODO:The procedure with Content-Encoding(RFC3261).
func (msg *sipMessage) parseSIPBody() (err error) {
	// in case no called before parseSIPHeader
	if (msg.headers) == nil {
		debugf("parseSIPBody: This sip message's .headers is nill.")
		return fmt.Errorf("headers is nill")
	}

	if msg.contentlength <= 0 {
		return nil
	}

	contenttypeArray, hdCtypeOk := (*msg.headers)["content-type"]

	// if not exist Content-Type header, return a error
	if !hdCtypeOk {
		debugf("parseSIPBody: This sip message has not body.")
		return fmt.Errorf("no content-type header")
	}

	msg.body = map[string]*map[string][]common.NetString{}

	if len(contenttypeArray) == 0 {
		//contenttypeArray has no element
		return nil
	}

	// Switch the function with body content type
	// TODO: Now, it is supported only SDP,
	//       more SIP body application support (I planning to support SIP-I(ISUP)/Multi-part)
	lowerCaseContentType := strings.ToLower(string(getLastElementStrArray(contenttypeArray)))
	switch lowerCaseContentType {
	case "application/sdp":
		body, err := msg.parseBodySDP(msg.raw[msg.bdyStart : msg.bdyStart+msg.contentlength])
		_ = err
		if err != nil {
			debugf("%s : parseError", lowerCaseContentType)
			return fmt.Errorf("invalid %s format", lowerCaseContentType)
		}

		msg.body[lowerCaseContentType] = body

	default:
		debugf("unsupported content-type. : %s", lowerCaseContentType)
		return fmt.Errorf("unsupported content-type")
	}

	return nil
}

func (msg *sipMessage) parseBodySDP(rawData []byte) (body *map[string][]common.NetString, err error) {
	body = &map[string][]common.NetString{}
	sdpLines := strings.Split(string(rawData), "\r\n")
	for i := 0; i < len(sdpLines); i++ {

		keyVal := strings.SplitN(sdpLines[i], "=", 2)

		if len(keyVal) != 2 {
			continue
		}

		key := strings.TrimSpace(keyVal[0])
		val := strings.TrimSpace(keyVal[1])

		_, existkey := (*body)[key]
		if !existkey {
			(*body)[key] = []common.NetString{}
		}
		(*body)[key] = append((*body)[key], common.NetString(val))
	}

	return body, nil
}

func (msg *sipMessage) getMessageStatus() int {
	if msg.isIncompletedHdrMsg {
		return SipStatusHeaderReceiving
	}
	if msg.isIncompletedBdyMsg {
		return SipStatusBodyReceiving
	}
	if msg.hdrStart < 0 {
		return SipStatusRejected
	}
	if msg.contentlength < 0 {
		return SipStatusBodyReceiving
	}
	return SipStatusReceived
}
