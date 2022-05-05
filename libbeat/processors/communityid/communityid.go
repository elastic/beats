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

package communityid

import (
	"crypto"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/flowhash"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	cfg "github.com/elastic/elastic-agent-libs/config"
)

const logName = "processor.community_id"

func init() {
	processors.RegisterPlugin("community_id", New)
	jsprocessor.RegisterPlugin("CommunityID", New)
}

type processor struct {
	config
	log    *logp.Logger
	hasher flowhash.Hasher
}

// New constructs a new processor that computes community ID flowhash. The
// values that are incorporated into the hash vary by protocol.
//
// TCP / UDP / SCTP:
//   IP src / IP dst / IP proto / source port / dest port
//
// ICMPv4 / ICMPv6:
//   IP src / IP dst / IP proto / ICMP type + "counter-type" or code
//
// Other IP-borne protocols:
//   IP src / IP dst / IP proto
func New(cfg *cfg.C) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the community_id configuration")
	}

	return newFromConfig(c)
}

func newFromConfig(c config) (*processor, error) {
	hasher := flowhash.CommunityID
	if c.Seed != 0 {
		hasher = flowhash.NewCommunityID(c.Seed, flowhash.Base64Encoding, crypto.SHA1)
	}

	return &processor{
		config: c,
		log:    logp.NewLogger(logName),
		hasher: hasher,
	}, nil
}

func (p *processor) String() string {
	return fmt.Sprintf("community_id=[target=%s, fields=["+
		"source_ip=%v, source_port=%v, "+
		"destination_ip=%v, destination_port=%v, "+
		"transport_protocol=%v, "+
		"icmp_type=%v, icmp_code=%v], seed=%d]",
		p.Target, p.Fields.SourceIP, p.Fields.SourcePort,
		p.Fields.DestinationIP, p.Fields.DestinationPort,
		p.Fields.TransportProtocol, p.Fields.ICMPType, p.Fields.ICMPCode,
		p.Seed)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	// If already set then bail out.
	_, err := event.GetValue(p.Target)
	if err == nil {
		return event, nil
	}

	flow := p.buildFlow(event)
	if flow == nil {
		return event, nil
	}

	id := p.hasher.Hash(*flow)
	_, err = event.PutValue(p.Target, id)
	return event, err
}

func (p *processor) buildFlow(event *beat.Event) *flowhash.Flow {
	var flow flowhash.Flow

	// source ip
	v, err := event.GetValue(p.Fields.SourceIP)
	if err != nil {
		return nil
	}
	var ok bool
	flow.SourceIP, ok = tryToIP(v)
	if !ok {
		return nil
	}

	// destination ip
	v, err = event.GetValue(p.Fields.DestinationIP)
	if err != nil {
		return nil
	}
	flow.DestinationIP, ok = tryToIP(v)
	if !ok {
		return nil
	}

	// protocol (try IANA number first)
	v, err = event.GetValue(p.Fields.IANANumber)
	if err != nil {
		// Try transport protocol name next.
		v, err = event.GetValue(p.Fields.TransportProtocol)
		if err != nil {
			return nil
		}
	}
	flow.Protocol, ok = tryToIANATransportProtocol(v)
	if !ok {
		return nil
	}

	switch flow.Protocol {
	case tcpProtocol, udpProtocol, sctpProtocol:
		// source port
		v, err = event.GetValue(p.Fields.SourcePort)
		if err != nil {
			return nil
		}
		sp, ok := tryToUint(v)
		if !ok || sp < 1 || sp > 65535 {
			return nil
		}
		flow.SourcePort = uint16(sp)

		// destination port
		v, err = event.GetValue(p.Fields.DestinationPort)
		if err != nil {
			return nil
		}
		dp, ok := tryToUint(v)
		if !ok || dp < 1 || dp > 65535 {
			return nil
		}
		flow.DestinationPort = uint16(dp)
	case icmpProtocol, icmpIPv6Protocol:
		// Return a flow even if the ICMP type/code is unavailable.
		if t, c, ok := getICMPTypeCode(event, p.Fields.ICMPType, p.Fields.ICMPCode); ok {
			flow.ICMP.Type, flow.ICMP.Code = t, c
		}
	}

	return &flow
}

func getICMPTypeCode(event *beat.Event, typeField, codeField string) (t, c uint8, ok bool) {
	v, err := event.GetValue(typeField)
	if err != nil {
		return 0, 0, false
	}
	t, ok = tryToUint8(v)
	if !ok {
		return 0, 0, false
	}

	v, err = event.GetValue(codeField)
	if err != nil {
		return 0, 0, false
	}
	c, ok = tryToUint8(v)
	if !ok {
		return 0, 0, false
	}
	return t, c, true
}

func tryToIP(from interface{}) (net.IP, bool) {
	switch v := from.(type) {
	case net.IP:
		return v, true
	case string:
		ip := net.ParseIP(v)
		return ip, ip != nil
	default:
		return nil, false
	}
}

// tryToUint tries to coerce the given interface to an uint16. On success it
// returns the int value and true.
func tryToUint(from interface{}) (uint, bool) {
	switch v := from.(type) {
	case int:
		return uint(v), true
	case int8:
		return uint(v), true
	case int16:
		return uint(v), true
	case int32:
		return uint(v), true
	case int64:
		return uint(v), true
	case uint:
		return uint(v), true
	case uint8:
		return uint(v), true
	case uint16:
		return uint(v), true
	case uint32:
		return uint(v), true
	case uint64:
		return uint(v), true
	case string:
		num, err := strconv.ParseUint(v, 0, 64)
		if err != nil {
			return 0, false
		}
		return uint(num), true
	default:
		return 0, false
	}
}

func tryToUint8(from interface{}) (uint8, bool) {
	to, ok := tryToUint(from)
	return uint8(to), ok
}

const (
	icmpProtocol     uint8 = 1
	igmpProtocol     uint8 = 2
	tcpProtocol      uint8 = 6
	udpProtocol      uint8 = 17
	greProtocol      uint8 = 47
	icmpIPv6Protocol uint8 = 58
	eigrpProtocol    uint8 = 88
	ospfProtocol     uint8 = 89
	pimProtocol      uint8 = 103
	sctpProtocol     uint8 = 132
)

var transports = map[string]uint8{
	"icmp":      icmpProtocol,
	"igmp":      igmpProtocol,
	"tcp":       tcpProtocol,
	"udp":       udpProtocol,
	"gre":       greProtocol,
	"ipv6-icmp": icmpIPv6Protocol,
	"icmpv6":    icmpIPv6Protocol,
	"eigrp":     eigrpProtocol,
	"ospf":      ospfProtocol,
	"pim":       pimProtocol,
	"sctp":      sctpProtocol,
}

func tryToIANATransportProtocol(from interface{}) (uint8, bool) {
	switch v := from.(type) {
	case string:
		transport, found := transports[v]
		if !found {
			transport, found = transports[strings.ToLower(v)]
		}
		if found {
			return transport, found
		}
	}

	// Allow raw protocol numbers.
	return tryToUint8(from)
}
