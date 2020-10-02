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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/packetbeat/pb"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

var (
	debugf    = logp.MakeDebug("sip")
	detailedf = logp.MakeDebug("sipdetailed")
)

// SIP application level protocol analyser plugin.
type plugin struct {
	// config
	ports              []int
	parseAuthorization bool
	parseBody          bool
	keepOriginal       bool

	results protos.Reporter
}

var (
	isDebug    = false
	isDetailed = false
)

func init() {
	protos.Register("sip", New)
}

func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	cfgwarn.Beta("packetbeat SIP protocol is used")

	p := &plugin{}
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
func (p *plugin) init(results protos.Reporter, config *config) error {
	p.setFromConfig(config)

	isDebug = logp.IsDebug("sip")
	isDetailed = logp.IsDebug("sipdetailed")
	p.results = results
	return nil
}

func (p *plugin) setFromConfig(config *config) {
	p.ports = config.Ports
	p.keepOriginal = config.KeepOriginal
	p.parseAuthorization = config.ParseAuthorization
	p.parseBody = config.ParseBody
}

func (p *plugin) GetPorts() []int {
	return p.ports
}

func (p *plugin) ParseUDP(pkt *protos.Packet) {
	defer logp.Recover("SIP ParseUDP exception")

	if err := p.doParse(pkt); err != nil {
		debugf("error: %s", err)
	}
}

func (p *plugin) doParse(pkt *protos.Packet) error {
	if isDetailed {
		detailedf("Payload received: [%s]", pkt.Payload)
	}

	parser := newParser()

	pi := newParsingInfo(pkt)
	m, err := parser.parse(pi)
	if err != nil {
		return err
	}

	p.publish(p.buildEvent(m, pkt))

	return nil
}

func (p *plugin) publish(evt beat.Event) {
	if p.results != nil {
		p.results(evt)
	}
}

func newParsingInfo(pkt *protos.Packet) *parsingInfo {
	return &parsingInfo{
		tuple: &pkt.Tuple,
		data:  pkt.Payload,
		pkt:   pkt,
	}
}

func (p *plugin) buildEvent(m *message, pkt *protos.Packet) beat.Event {
	status := common.OK_STATUS
	if m.statusCode >= 400 {
		status = common.ERROR_STATUS
	}

	evt, pbf := pb.NewBeatEvent(m.ts)
	fields := evt.Fields
	fields["type"] = "sip"
	fields["status"] = status

	var sipFields ProtocolFields
	sipFields.Timestamp = time.Now().UnixNano()
	if m.isRequest {
		populateRequestFields(m, pbf, &sipFields)
	} else {
		populateResponseFields(m, &sipFields)
	}

	p.populateHeadersFields(m, pbf, &sipFields)

	if p.parseBody {
		populateBodyFields(m, pbf, &sipFields)
	}

	pbf.Network.IANANumber = "17"
	pbf.Network.Application = "sip"
	pbf.Network.Protocol = "sip"
	pbf.Network.Transport = "udp"

	src, dst := m.getEndpoints()
	pbf.SetSource(src)
	pbf.SetDestination(dst)

	p.populateEventFields(m, pbf, sipFields)

	_ = pb.MarshalStruct(evt.Fields, "sip", sipFields)

	return evt
}

func populateRequestFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	fields.Type = "request"
	fields.Method = bytes.ToUpper(m.method)
	fields.URIOriginal = m.requestURI
	scheme, username, host, port := parseURI(fields.URIOriginal)
	fields.URIScheme = scheme
	fields.URIHost = host
	fields.URIUsername = username
	fields.URIPort = port
	fields.Version = m.version.String()
	pbf.AddHost(string(host))
	pbf.AddUser(string(username))
}

func populateResponseFields(m *message, fields *ProtocolFields) {
	fields.Type = "response"
	fields.Code = int(m.statusCode)
	fields.Status = m.statusPhrase
	fields.Version = m.version.String()
}

func (p *plugin) populateHeadersFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	accept, found := m.headers["accept"]
	if found && len(accept) > 0 {
		fields.Accept = bytes.ToLower(accept[0])
	}
	fields.Allow = m.allow
	fields.CallID = m.callID
	fields.ContentLength = m.contentLength
	fields.ContentType = bytes.ToLower(m.contentType)
	fields.MaxForwards = m.maxForwards
	privateURI, found := m.headers["p-associated-uri"]
	if found && len(privateURI) > 0 {
		scheme, username, host, port := parseURI(privateURI[0])
		fields.PrivateURIOriginal = privateURI[0]
		fields.PrivateURIScheme = scheme
		fields.PrivateURIHost = host
		fields.PrivateURIUsername = username
		fields.PrivateURIPort = port
		pbf.AddHost(string(host))
		pbf.AddUser(string(username))
	}
	fields.Supported = m.supported
	fields.UserAgentOriginal = m.userAgent

	cseqParts := bytes.Split(m.cseq, []byte(" "))
	if len(cseqParts) == 2 {
		fields.CseqCode, _ = strconv.Atoi(string(cseqParts[0]))
		fields.CseqMethod = bytes.ToUpper(cseqParts[1])
	}

	populateViaFields(m, pbf, fields)

	if len(m.from) > 0 {
		displayInfo, uri, tag := parseFromTo(m.from)
		fields.FromDisplayInfo = displayInfo
		fields.FromTag = tag
		scheme, username, host, port := parseURI(uri)
		fields.FromURIOriginal = uri
		fields.FromURIScheme = scheme
		fields.FromURIHost = host
		fields.FromURIUsername = username
		fields.FromURIPort = port
		pbf.AddHost(string(host))
		pbf.AddUser(string(username))
	}

	if len(m.to) > 0 {
		displayInfo, uri, tag := parseFromTo(m.to)
		fields.ToDisplayInfo = displayInfo
		fields.ToTag = tag
		scheme, username, host, port := parseURI(uri)
		fields.ToURIOriginal = uri
		fields.ToURIScheme = scheme
		fields.ToURIHost = host
		fields.ToURIUsername = username
		fields.ToURIPort = port
		pbf.AddHost(string(host))
		pbf.AddUser(string(username))
	}

	populateContactFields(m, pbf, fields)

	if p.parseAuthorization {
		populateAuthFields(m, pbf, fields)
	}
}

func (p *plugin) populateEventFields(m *message, pbf *pb.Fields, sipFields ProtocolFields) {
	pbf.Event.Kind = "event"
	pbf.Event.Type = []string{"info"}
	pbf.Event.Dataset = "sip"
	pbf.Event.Sequence = int64(sipFields.CseqCode)

	// TODO: Get these values from body
	pbf.Event.Start = m.ts
	pbf.Event.End = m.ts
	//

	if p.keepOriginal {
		pbf.Event.Original = string(m.rawData)
	}

	pbf.Event.Category = []string{"network", "protocol"}
	if _, found := m.headers["authorization"]; found {
		pbf.Event.Category = append(pbf.Event.Category, "authorization")
	}

	pbf.Event.Action = func() string {
		if m.isRequest {
			return fmt.Sprintf("sip_%s", strings.ToLower(string(m.method)))
		}
		return fmt.Sprintf("sip_%s", strings.ToLower(string(sipFields.CseqMethod)))
	}()

	pbf.Event.Outcome = func() string {
		switch {
		case m.statusCode < 200:
			return ""
		case m.statusCode > 299:
			return "failure"
		}
		return "success"
	}()

	pbf.Event.Reason = string(sipFields.Status)
}

func populateViaFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	// TODO
}

func populateAuthFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	// TODO
}

func populateContactFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	// TODO
}

func populateBodyFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	// TODO
}

func parseFromTo(fromTo common.NetString) (displayInfo, uri, tag common.NetString) {
	spacePos := bytes.IndexByte(fromTo, '<')
	if spacePos == -1 {
		return nil, nil, nil
	}
	if spacePos > 0 {
		spacePos -= 1
	}
	displayInfo = bytes.Trim(fromTo[:spacePos], `'"`)
	parts := bytes.Split(fromTo[spacePos+1:], []byte(";"))
	uri = bytes.Trim(parts[0], "<>")
	if len(parts) == 2 {
		tag = bytes.TrimSpace(parts[1])[len("tag="):]
	}
	return displayInfo, uri, tag
}

func parseURI(uri common.NetString) (scheme, username, host common.NetString, port int) {
	var prevChar rune
	uri = bytes.TrimSpace(uri)
	prevChar = ' '
	pos := -1
	ppos := -1
	epos := len(uri)
	inIPv6 := false
	idx := 0

loop:
	for idx = 0; idx < len(uri); idx++ {
		curChar := rune(uri[idx])

		switch {
		case idx == 0:
			colonIdx := bytes.Index(uri, []byte(":"))
			if colonIdx == -1 {
				break loop
			}
			scheme = uri[:colonIdx]
			idx += colonIdx
			pos = idx + 1

		case curChar == '[' && prevChar != '\\':
			inIPv6 = true

		case curChar == ']' && prevChar != '\\':
			inIPv6 = false

		case curChar == ';' && prevChar != '\\':
			// we found end of URI and will ignore extra info
			epos = idx
			break loop

		default:
			// select wich part
			switch curChar {
			case '@':
				if len(host) > 0 {
					pos = ppos
					host = nil
				}
				username = uri[pos:idx]
				ppos = pos
				pos = idx + 1
			case ':':
				if !inIPv6 {
					host = uri[pos:idx]
					ppos = pos
					pos = idx + 1
				}
			}
		}

		prevChar = curChar
	}

	if pos > 0 && epos <= len(uri) && pos <= epos {
		if len(host) == 0 {
			host = bytes.TrimSpace(uri[pos:epos])
		} else {
			port, _ = strconv.Atoi(string(bytes.TrimSpace(uri[pos:epos])))
		}
	}

	return scheme, username, host, port
}
