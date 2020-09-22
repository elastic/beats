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
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
)

type sipPlugin struct {
	// Configuration data.
	ports               []int
	includeRawMessage   bool
	includeHeaders      bool
	includeBody         bool
	parseDetail         bool
	useDefaultHeaders   bool
	headersToParseAsURI []string
	headersToParseAsInt []string
	parseSet            map[string]int

	results protos.Reporter // Channel where results are pushed.
}

func (sip *sipPlugin) init(results protos.Reporter, config *sipConfig) error {
	sip.setFromConfig(config)

	if sip.parseDetail {
		sip.initDetailOption()
	}

	sip.results = results

	return nil
}

func (sip *sipPlugin) initDetailOption() {
	// Detail of headers
	sip.parseSet = make(map[string]int)

	if sip.useDefaultHeaders {
		sip.parseSet["from"] = detailNameAddr
		sip.parseSet["to"] = detailNameAddr
		sip.parseSet["contact"] = detailNameAddr
		sip.parseSet["record-route"] = detailNameAddr
		sip.parseSet["p-asserted-identity"] = detailNameAddr
		sip.parseSet["p-preferred-identity"] = detailNameAddr
	}
	for _, header := range sip.headersToParseAsURI {
		header = strings.ToLower(strings.TrimSpace(header))
		sip.parseSet[header] = detailNameAddr
	}
	sip.parseSet["cseq"] = detailIntMethod
	sip.parseSet["rack"] = detailIntIntMethod
	if sip.useDefaultHeaders {
		sip.parseSet["rseq"] = detailInt
		sip.parseSet["content-length"] = detailInt
		sip.parseSet["max-forwards"] = detailInt
		sip.parseSet["expires"] = detailInt
		sip.parseSet["session-expires"] = detailInt
		sip.parseSet["min-se"] = detailInt
	}
	for _, header := range sip.headersToParseAsInt {
		header = strings.ToLower(strings.TrimSpace(header))
		sip.parseSet[header] = detailInt
	}
}

// Set config values sip ports and options.
func (sip *sipPlugin) setFromConfig(config *sipConfig) {
	sip.ports = config.Ports
	sip.includeRawMessage = config.IncludeRawMessage
	sip.includeHeaders = config.IncludeHeaders
	sip.includeBody = config.IncludeBody
	sip.parseDetail = config.ParseDetail
	sip.useDefaultHeaders = config.UseDefaultHeaders
	sip.headersToParseAsURI = config.HeadersToParseAsURI
	sip.headersToParseAsInt = config.HeadersToParseAsInt
}

// Getter : instance Ports int slice
func (sip *sipPlugin) GetPorts() []int {
	return sip.ports
}

// publishMessage to reshape the sipMessage for in order to pushing with json.
func (sip *sipPlugin) publishMessage(msg *sipMessage) {
	if sip.results == nil {
		return
	}

	debugf("Publishing SIP Message. %s", msg.String())

	timestamp := msg.ts
	fields := common.MapStr{}

	fields.Put("type", "sip")
	fields.Put("event.dataset", "sip")

	fields.Put("event.kind", "event")
	fields.Put("event.category", []string{"network"})
	if sip.includeRawMessage {
		fields.Put("event.original", string(msg.raw))
	}

	fields.Put("source.ip", msg.tuple.SrcIP)
	fields.Put("source.port", msg.tuple.SrcPort)

	fields.Put("destination.ip", msg.tuple.DstIP)
	fields.Put("destination.port", msg.tuple.DstPort)

	fields.Put("related.ip", []net.IP{msg.tuple.SrcIP, msg.tuple.DstIP})

	fields.Put("network.transport", msg.transport.String())

	sipFields := common.MapStr{}
	fields.Put("sip", sipFields)

	sipFields.Put("unixtimenano", time.Now().UnixNano())

	if msg.isRequest {
		sipFields.Put("method", string(msg.method))
		sipFields.Put("request_uri", string(msg.requestURI))
	} else {
		sipFields.Put("status.code", int(msg.statusCode))
		sipFields.Put("status.phrase", string(msg.statusPhrase))
	}

	sipFields.Put("from", string(msg.from))
	sipFields.Put("to", string(msg.to))
	sipFields.Put("cseq", string(msg.cseq))
	sipFields.Put("call_id", string(msg.callid))

	sipHeaders := common.MapStr{}
	if sip.includeHeaders {
		sipFields["headers"] = sipHeaders

		if msg.headers != nil {
			for header, lines := range *(msg.headers) {
				sipHeaders[header] = lines
			}
		}
	}

	if sip.includeBody {
		sipBody := common.MapStr{}
		sipFields["body"] = sipBody

		if msg.body != nil {
			for content, keyval := range msg.body {
				contetMap := common.MapStr{}
				sipBody[content] = contetMap
				for key, valLines := range *keyval {
					contetMap[key] = valLines
				}
			}
		}
	}

	if sip.parseDetail {
		var displayName, userInfo, host, port string
		var addrparams, params []string
		var number int
		var err error

		// Detail of Request-URI
		if value, ok := sipFields["request_uri"]; ok {
			userInfo, host, port, addrparams = sip.parseDetailURI(value.(string))

			sipFields["request_uri_user"] = userInfo
			number, err = strconv.Atoi(strings.TrimSpace(port))
			if err == nil {
				sipFields["request_uri_port"] = number
			}
			sipFields["request_uri_host"] = host
			if len(addrparams) > 0 {
				sipFields["request_uri_params"] = addrparams
			}
		}

		for key, values := range sipHeaders {
			newval := []common.MapStr{}

			for _, headerS := range values.([]common.NetString) {
				newobj := common.MapStr{}
				newobj["raw"] = headerS

				if mode, ok := sip.parseSet[key]; ok {
					switch mode {
					case detailNameAddr:
						displayName, userInfo, host, port, addrparams, params = sip.parseDetailNameAddr(fmt.Sprintf("%s", headerS))

						number, err = strconv.Atoi(port)
						if displayName != "" {
							newobj["display"] = displayName
						}
						if userInfo != "" {
							newobj["user"] = userInfo
						}
						if host != "" {
							newobj["host"] = host
						}
						if err == nil {
							newobj["port"] = number
						}
						if len(addrparams) > 0 {
							newobj["uri_params"] = addrparams
						}
						if len(params) > 0 {
							newobj["params"] = params
						}

					case detailInt:
						number, err = strconv.Atoi(strings.TrimSpace(string(headerS)))
						if err == nil {
							newobj["number"] = number
						}

					case detailIntMethod:
						values := strings.SplitN(string(headerS), " ", 2)
						number, err = strconv.Atoi(strings.TrimSpace(values[0]))
						if err == nil {
							newobj["number"] = number
						}
						newobj["method"] = strings.TrimSpace(values[1])

					case detailIntIntMethod:
						values := strings.SplitN(string(headerS), " ", 3)
						number, err = strconv.Atoi(strings.TrimSpace(values[0]))
						if err == nil {
							newobj["number1"] = number
						}
						number, err = strconv.Atoi(strings.TrimSpace(values[1]))
						if err == nil {
							newobj["number2"] = number
						}
						newobj["method"] = strings.TrimSpace(values[2])
					}
				}
				newval = append(newval, newobj)
			}
			sipHeaders[key] = newval
		}
	}

	if msg.notes != nil {
		fields["notes"] = fmt.Sprintf("%s", msg.notes)
	}

	sip.results(beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	})
}

// createSIPMessage a byte array into a SIP struct. If an error occurs
// then the returned sip pointer will be nil. This method recovers from panics
// and is concurrency-safe.
func (sip *sipPlugin) createSIPMessage(transp transport, rawData []byte) (msg *sipMessage, err error) {
	// Recover from any panics that occur while parsing a packet.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	// create and initialized pakcet raw message and transport type.
	msg = &sipMessage{}
	msg.transport = transp
	msg.raw = rawData

	// offset values are initialized to -1
	msg.hdrStart = -1
	msg.hdrLen = -1
	msg.bdyStart = -1
	msg.contentlength = -1

	msg.isIncompletedHdrMsg = false
	msg.isIncompletedBdyMsg = false

	return msg, nil
}

func (sip *sipPlugin) ParseUDP(pkt *protos.Packet) {

	defer logp.Recover("Sip ParseUdp")
	packetSize := len(pkt.Payload)

	debugf("Parsing packet addressed with %s of length %d.", pkt.Tuple.String(), packetSize)

	var sipMsg *sipMessage
	var err error

	debugf("New sip message: %s %s", &pkt.Tuple, transportUDP)

	// create new SIP Message
	sipMsg, err = sip.createSIPMessage(transportUDP, pkt.Payload)

	if err != nil {
		// ignore this message
		debugf("error %s\n", err)
		return
	}

	sipMsg.ts = pkt.Ts
	sipMsg.tuple = pkt.Tuple
	sipMsg.cmdlineTuple = procs.ProcWatcher.FindProcessesTuple(&pkt.Tuple, applayer.TransportUDP)

	// parse sip headers.
	// if the message was malformed, the message will be rejected
	parseHeaderErr := sipMsg.parseSIPHeader()
	if parseHeaderErr != nil {
		debugf("error %s\n", parseHeaderErr)
		return
	}

	switch sipMsg.getMessageStatus() {
	case statusRejected:
		return
	// In case the message was incompleted at header or body,
	// the message was added error notes and published.
	case statusHeaderReceiving, statusBodyReceiving:
		debugf("Incompleted message")
		sipMsg.notes = append(sipMsg.notes, common.NetString("Incompleted message"))

	// In case the message received completely, publishing the message.
	case statusReceived:
		err := sipMsg.parseSIPBody()
		if err != nil {
			sipMsg.notes = append(sipMsg.notes, common.NetString(fmt.Sprintf("%s", err)))
		}
	}
	sip.publishMessage(sipMsg)
}

func (sip *sipPlugin) parseDetailURI(addr string) (userInfo string, host string, port string, params []string) {
	var prevChar rune
	addr = strings.TrimSpace(addr)
	prevChar = ' '
	pos := -1
	ppos := -1
	epos := len(addr)
	inIPv6 := false
	idx := 0
	for idx = 0; idx < len(addr); idx++ {
		curChar := rune(addr[idx])

		if idx == 0 {
			if idx+4 >= len(addr) {
				break
			}
			// sip/sips/tel-uri
			if addr[idx:idx+5] == "sips:" {
				idx += 4
			} else if addr[idx:idx+4] == "sip:" || addr[idx:idx+4] == "tel:" {
				idx += 3
			} else {
				break
			}
			pos = idx + 1
		} else if curChar == '[' && prevChar != '\\' {
			inIPv6 = true
		} else if curChar == ']' && prevChar != '\\' {
			inIPv6 = false
		} else if curChar == ';' && prevChar != '\\' {
			if len(params) == 0 {
				epos = idx
				params = strings.Split(addr[idx+1:], ";")
			}
			//break
		} else {
			// select wich part
			switch curChar {
			case '@':
				if host != "" {
					pos = ppos
					host = ""
				}
				if len(params) > 0 {
					epos = len(addr)
					params = params[:0] // clear slice
				}
				userInfo = addr[pos:idx]
				ppos = pos
				pos = idx + 1
			case ':':
				if !inIPv6 {
					host = addr[pos:idx]
					ppos = pos
					pos = idx + 1
				}
			}
		}
		prevChar = curChar
	}
	if pos > 0 && epos <= len(addr) && pos <= epos {
		if host == "" {
			host = strings.TrimSpace(addr[pos:epos])
		} else {
			port = strings.TrimSpace(addr[pos:epos])
		}
	}

	return userInfo, host, port, params
}

func (sip *sipPlugin) parseDetailNameAddr(addr string) (displayName string, userInfo string, host string, port string, addrparams []string, params []string) {

	addr = strings.TrimSpace(addr)
	var prevChar rune
	prevChar = ' '
	pos := -1
	_ = port
	inAddr := false
	escaped := false

	for idx := 0; idx < len(addr); idx++ {
		curChar := rune(addr[idx])
		// Display name
		if !inAddr && displayName == "" && userInfo == "" && host == "" {
			if idx == 0 && idx+5 < len(addr) {
				if addr[idx:idx+5] == "sips:" || addr[idx:idx+4] == "sip:" || addr[idx:idx+4] == "tel:" {
					userInfo, host, port, addrparams = sip.parseDetailURI(addr[idx:])
					idx = len(addr)
					break
				}
			}
			if idx == 0 && curChar != '<' {
				pos = idx
				if curChar == '"' {
					pos++
					escaped = true
				}
				continue
			} else if curChar == '"' && prevChar != '\\' {
				displayName = addr[pos:idx]
				pos = -1
			} else if escaped {
				prevChar = curChar
				continue
			}
		}
		if curChar == '<' && !inAddr && prevChar != '\\' {
			if displayName == "" && pos >= 0 {
				displayName = strings.TrimSpace(addr[pos:idx])
			}
			pos = idx + 1
			for idx = idx + 1; idx < len(addr); idx++ {
				if rune(addr[idx]) == '>' && addr[idx-1] != '\\' {
					userInfo, host, port, addrparams = sip.parseDetailURI(addr[pos:idx])

					for idx = idx + 1; idx < len(addr); idx++ {
						if rune(addr[idx]) == ';' {
							substr := addr[idx+1:]
							params = strings.Split(substr, ";")
							idx = len(addr)
						}
					}
					break
				}
			}
		}

		prevChar = curChar
	}

	return displayName, userInfo, host, port, addrparams, params
}
