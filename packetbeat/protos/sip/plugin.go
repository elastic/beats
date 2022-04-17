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

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/cfgwarn"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/packetbeat/pb"
	"github.com/menderesk/beats/v7/packetbeat/procs"
	"github.com/menderesk/beats/v7/packetbeat/protos"
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
	watcher procs.ProcessesWatcher
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
	watcher procs.ProcessesWatcher,
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

	if err := p.init(results, watcher, &config); err != nil {
		return nil, err
	}
	return p, nil
}

// Init initializes the HTTP protocol analyser.
func (p *plugin) init(results protos.Reporter, watcher procs.ProcessesWatcher, config *config) error {
	p.setFromConfig(config)

	isDebug = logp.IsDebug("sip")
	isDetailed = logp.IsDebug("sipdetailed")
	p.results = results
	p.watcher = watcher
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

	parser := newParser(p.watcher)

	pi := newParsingInfo(pkt)
	m, err := parser.parse(pi)
	if err != nil {
		return err
	}

	evt, err := p.buildEvent(m, pkt)
	if err != nil {
		return err
	}

	p.publish(*evt)

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

func (p *plugin) buildEvent(m *message, pkt *protos.Packet) (*beat.Event, error) {
	status := common.OK_STATUS
	if m.statusCode >= 400 {
		status = common.ERROR_STATUS
	}

	evt, pbf := pb.NewBeatEvent(m.ts)
	fields := evt.Fields
	fields["type"] = "sip"
	fields["status"] = status

	var sipFields ProtocolFields
	if m.isRequest {
		populateRequestFields(m, pbf, &sipFields)
	} else {
		populateResponseFields(m, &sipFields)
	}

	p.populateHeadersFields(m, evt, pbf, &sipFields)

	if p.parseBody {
		populateBodyFields(m, pbf, &sipFields)
	}

	pbf.Network.IANANumber = "17"
	pbf.Network.Application = "sip"
	pbf.Network.Protocol = "sip"
	pbf.Network.Transport = "udp"

	src, dst := m.getEndpoints()
	pbf.SetSource(src)
	pbf.AddIP(src.IP)
	pbf.SetDestination(dst)
	pbf.AddIP(dst.IP)

	p.populateEventFields(m, pbf, sipFields)

	if err := pb.MarshalStruct(evt.Fields, "sip", sipFields); err != nil {
		return nil, err
	}

	return &evt, nil
}

func populateRequestFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	fields.Type = "request"
	fields.Method = bytes.ToUpper(m.method)
	fields.URIOriginal = m.requestURI
	scheme, username, host, port, _ := parseURI(fields.URIOriginal)
	fields.URIScheme = scheme
	fields.URIHost = host
	if !bytes.Equal(username, []byte(" ")) && !bytes.Equal(username, []byte("-")) {
		fields.URIUsername = username
		pbf.AddUser(string(username))
	}
	fields.URIPort = port
	fields.Version = m.version.String()
	pbf.AddHost(string(host))
}

func populateResponseFields(m *message, fields *ProtocolFields) {
	fields.Type = "response"
	fields.Code = int(m.statusCode)
	fields.Status = m.statusPhrase
	fields.Version = m.version.String()
}

func (p *plugin) populateHeadersFields(m *message, evt beat.Event, pbf *pb.Fields, fields *ProtocolFields) {
	fields.Allow = m.allow
	fields.CallID = m.callID
	fields.ContentLength = m.contentLength
	fields.ContentType = bytes.ToLower(m.contentType)
	fields.MaxForwards = m.maxForwards
	fields.Supported = m.supported
	fields.UserAgentOriginal = m.userAgent
	fields.ViaOriginal = m.via

	privateURI, found := m.headers["p-associated-uri"]
	if found && len(privateURI) > 0 {
		scheme, username, host, port, _ := parseURI(privateURI[0])
		fields.PrivateURIOriginal = privateURI[0]
		fields.PrivateURIScheme = scheme
		fields.PrivateURIHost = host
		if !bytes.Equal(username, []byte(" ")) && !bytes.Equal(username, []byte("-")) {
			fields.PrivateURIUsername = username
			pbf.AddUser(string(username))
		}
		fields.PrivateURIPort = port
		pbf.AddHost(string(host))
	}

	if accept, found := m.headers["accept"]; found && len(accept) > 0 {
		fields.Accept = bytes.ToLower(accept[0])
	}

	cseqParts := bytes.Split(m.cseq, []byte(" "))
	if len(cseqParts) == 2 {
		fields.CseqCode, _ = strconv.Atoi(string(cseqParts[0]))
		fields.CseqMethod = bytes.ToUpper(cseqParts[1])
	}

	populateFromFields(m, pbf, fields)

	populateToFields(m, pbf, fields)

	populateContactFields(m, pbf, fields)

	if p.parseAuthorization {
		populateAuthFields(m, evt, pbf, fields)
	}
}

func populateFromFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	if len(m.from) > 0 {
		displayInfo, uri, params := parseFromToContact(m.from)
		fields.FromDisplayInfo = displayInfo
		fields.FromTag = params["tag"]
		scheme, username, host, port, _ := parseURI(uri)
		fields.FromURIOriginal = uri
		fields.FromURIScheme = scheme
		fields.FromURIHost = host
		if !bytes.Equal(username, []byte(" ")) && !bytes.Equal(username, []byte("-")) {
			fields.FromURIUsername = username
			pbf.AddUser(string(username))
		}
		fields.FromURIPort = port
		pbf.AddHost(string(host))
	}
}

func populateToFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	if len(m.to) > 0 {
		displayInfo, uri, params := parseFromToContact(m.to)
		fields.ToDisplayInfo = displayInfo
		fields.ToTag = params["tag"]
		scheme, username, host, port, _ := parseURI(uri)
		fields.ToURIOriginal = uri
		fields.ToURIScheme = scheme
		fields.ToURIHost = host
		if !bytes.Equal(username, []byte(" ")) && !bytes.Equal(username, []byte("-")) {
			fields.ToURIUsername = username
			pbf.AddUser(string(username))
		}
		fields.ToURIPort = port
		pbf.AddHost(string(host))
	}
}

func populateContactFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	if contact, found := m.headers["contact"]; found && len(contact) > 0 {
		displayInfo, uri, params := parseFromToContact(m.to)
		fields.ContactDisplayInfo = displayInfo
		fields.ContactExpires, _ = strconv.Atoi(string(params["expires"]))
		fields.ContactQ, _ = strconv.ParseFloat(string(params["q"]), 64)
		scheme, username, host, port, urlparams := parseURI(uri)
		fields.ContactURIOriginal = uri
		fields.ContactURIScheme = scheme
		fields.ContactURIHost = host
		if !bytes.Equal(username, []byte(" ")) && !bytes.Equal(username, []byte("-")) {
			fields.ContactURIUsername = username
			pbf.AddUser(string(username))
		}
		fields.ContactURIPort = port
		fields.ContactLine = urlparams["line"]
		fields.ContactTransport = bytes.ToLower(urlparams["transport"])
		pbf.AddHost(string(host))
	}
}

func (p *plugin) populateEventFields(m *message, pbf *pb.Fields, fields ProtocolFields) {
	pbf.Event.Kind = "event"
	pbf.Event.Type = []string{"info"}
	pbf.Event.Dataset = "sip"
	pbf.Event.Sequence = int64(fields.CseqCode)

	// TODO: Get these values from body
	pbf.Event.Start = m.ts
	pbf.Event.End = m.ts
	//

	if p.keepOriginal {
		pbf.Event.Original = string(m.rawData)
	}

	pbf.Event.Category = []string{"network", "protocol"}
	if _, found := m.headers["authorization"]; found {
		pbf.Event.Category = append(pbf.Event.Category, "authentication")
	}

	pbf.Event.Action = func() string {
		if m.isRequest {
			return fmt.Sprintf("sip-%s", strings.ToLower(string(m.method)))
		}
		return fmt.Sprintf("sip-%s", strings.ToLower(string(fields.CseqMethod)))
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

	pbf.Event.Reason = string(fields.Status)
}

func populateAuthFields(m *message, evt beat.Event, pbf *pb.Fields, fields *ProtocolFields) {
	auths, found := m.headers["authorization"]
	if !found || len(auths) == 0 {
		if isDetailed {
			detailedf("sip packet without authorization header")
		}
		return
	}

	if isDetailed {
		detailedf("sip packet with authorization header")
	}

	auth := bytes.TrimSpace(auths[0])
	pos := bytes.IndexByte(auth, ' ')
	if pos == -1 {
		if isDebug {
			debugf("malformed authorization header: missing scheme")
		}
		return
	}

	fields.AuthScheme = auth[:pos]

	pos += 1
	for _, param := range bytes.Split(auth[pos:], []byte(",")) {
		kv := bytes.SplitN(param, []byte("="), 2)
		if len(kv) != 2 {
			continue
		}
		kv[1] = bytes.Trim(kv[1], "'\" \t")
		switch string(bytes.ToLower(bytes.TrimSpace(kv[0]))) {
		case "realm":
			fields.AuthRealm = kv[1]
		case "username":
			username := string(kv[1])
			if username != "" && username != "-" {
				_, _ = evt.Fields.Put("user.name", username)
				pbf.AddUser(username)
			}
		case "uri":
			scheme, _, host, port, _ := parseURI(kv[1])
			fields.AuthURIOriginal = kv[1]
			fields.AuthURIScheme = scheme
			fields.AuthURIHost = host
			fields.AuthURIPort = port
		}
	}
}

var constSDPContentType = []byte("application/sdp")

func populateBodyFields(m *message, pbf *pb.Fields, fields *ProtocolFields) {
	if !m.hasContentLength {
		return
	}

	if !bytes.Equal(m.contentType, constSDPContentType) {
		if isDebug {
			debugf("body content-type: %s is not supported", m.contentType)
		}
		return
	}

	if _, found := m.headers["content-encoding"]; found {
		if isDebug {
			debugf("body decoding is not supported yet if content-endcoding is present")
		}
		return
	}

	fields.SDPBodyOriginal = m.body

	var isInMedia bool
	for _, line := range bytes.Split(m.body, []byte("\r\n")) {
		kv := bytes.SplitN(line, []byte("="), 2)
		if len(kv) != 2 {
			continue
		}

		kv[1] = bytes.TrimSpace(kv[1])
		ch := string(bytes.ToLower(bytes.TrimSpace(kv[0])))
		switch ch {
		case "v":
			fields.SDPVersion = string(kv[1])
		case "o":
			var pos int
			if kv[1][pos] == '"' {
				endUserPos := bytes.IndexByte(kv[1][pos+1:], '"')
				if !bytes.Equal(kv[1][pos+1:endUserPos], []byte("-")) {
					fields.SDPOwnerUsername = kv[1][pos+1 : endUserPos]
				}
				pos = endUserPos + 1
			}
			nParts := func() int {
				if pos == 0 {
					return 4
				}
				return 3 // already have user
			}()
			parts := bytes.SplitN(kv[1][pos:], []byte(" "), nParts)
			if len(parts) != nParts {
				if isDebug {
					debugf("malformed owner SDP line")
				}
				continue
			}
			if nParts == 4 {
				if !bytes.Equal(parts[0], []byte("-")) {
					fields.SDPOwnerUsername = parts[0]
				}
				parts = parts[1:]
			}
			fields.SDPOwnerSessID = parts[0]
			fields.SDPOwnerVersion = parts[1]
			fields.SDPOwnerIP = func() common.NetString {
				p := bytes.Split(parts[2], []byte(" "))
				return p[len(p)-1]
			}()
			pbf.AddUser(string(fields.SDPOwnerUsername))
			pbf.AddIP(string(fields.SDPOwnerIP))
		case "s":
			if !bytes.Equal(kv[1], []byte("-")) {
				fields.SDPSessName = kv[1]
			}
		case "c":
			if isInMedia {
				continue
			}
			fields.SDPConnInfo = kv[1]
			fields.SDPConnAddr = func() common.NetString {
				p := bytes.Split(kv[1], []byte(" "))
				return p[len(p)-1]
			}()
			pbf.AddHost(string(fields.SDPConnAddr))
		case "m":
			isInMedia = true
			// TODO
		case "i", "u", "e", "p", "b", "t", "r", "z", "k", "a":
			// TODO
		}
	}
}

func parseFromToContact(fromTo common.NetString) (displayInfo, uri common.NetString, params map[string]common.NetString) {
	params = make(map[string]common.NetString)

	fromTo = bytes.TrimSpace(fromTo)

	var uriIsWrapped bool
	pos := func() int {
		// look for the beginning of a url wrapped in <...>
		if pos := bytes.IndexByte(fromTo, '<'); pos > -1 {
			uriIsWrapped = true
			return pos
		}
		// if there is no < char, it means there is no display info, and
		// that the url starts from the beginning
		// https://tools.ietf.org/html/rfc3261#section-20.10
		return 0
	}()

	displayInfo = bytes.Trim(fromTo[:pos], "'\"\t ")

	endURIPos := func() int {
		if uriIsWrapped {
			return bytes.IndexByte(fromTo, '>')
		}
		return bytes.IndexByte(fromTo, ';')
	}()

	// not wrapped and no header params
	if endURIPos == -1 {
		uri = fromTo[pos:]
		return displayInfo, uri, params
	}

	// if wrapped, we want to get over the < char
	if uriIsWrapped {
		pos += 1
	}

	// if wrapped, we will get the string between <...>
	// if not wrapped, we will get the value before the header params (until ;)
	uri = fromTo[pos:endURIPos]

	// parse the header params
	pos = endURIPos + 1
	for _, param := range bytes.Split(fromTo[pos:], []byte(";")) {
		kv := bytes.SplitN(param, []byte("="), 2)
		if len(kv) != 2 {
			continue
		}
		params[string(bytes.ToLower(bytes.TrimSpace(kv[0])))] = kv[1]
	}

	return displayInfo, uri, params
}

func parseURI(uri common.NetString) (scheme, username, host common.NetString, port int, params map[string]common.NetString) {
	var (
		prevChar  rune
		inIPv6    bool
		idx       int
		hasParams bool
	)
	uri = bytes.TrimSpace(uri)
	prevChar = ' '
	pos := -1
	ppos := -1
	epos := len(uri)

	params = make(map[string]common.NetString)
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
			// we found end of URI
			hasParams = true
			epos = idx
			break loop

		default:
			// select which part
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

	if hasParams {
		for _, param := range bytes.Split(uri[epos+1:], []byte(";")) {
			kv := bytes.Split(param, []byte("="))
			if len(kv) != 2 {
				continue
			}
			params[string(bytes.ToLower(bytes.TrimSpace(kv[0])))] = kv[1]
		}
	}

	return scheme, username, host, port, params
}
