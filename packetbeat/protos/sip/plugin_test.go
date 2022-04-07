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

//go:build !integration
// +build !integration

package sip

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/packetbeat/procs"
	"github.com/elastic/beats/v8/packetbeat/protos"
)

func TestParseURI(t *testing.T) {
	scheme, username, host, port, params := parseURI(common.NetString("sip:test@10.0.2.15:5060"))
	assert.Equal(t, common.NetString("sip"), scheme)
	assert.Equal(t, common.NetString("test"), username)
	assert.Equal(t, common.NetString("10.0.2.15"), host)
	assert.Equal(t, map[string]common.NetString{}, params)
	assert.Equal(t, 5060, port)

	scheme, username, host, port, params = parseURI(common.NetString("sips:test@10.0.2.15:5061 ; transport=udp"))
	assert.Equal(t, common.NetString("sips"), scheme)
	assert.Equal(t, common.NetString("test"), username)
	assert.Equal(t, common.NetString("10.0.2.15"), host)
	assert.Equal(t, common.NetString("udp"), params["transport"])
	assert.Equal(t, 5061, port)

	scheme, username, host, port, params = parseURI(common.NetString("mailto:192.168.0.2"))
	assert.Equal(t, common.NetString("mailto"), scheme)
	assert.Equal(t, common.NetString(nil), username)
	assert.Equal(t, common.NetString("192.168.0.2"), host)
	assert.Equal(t, map[string]common.NetString{}, params)
	assert.Equal(t, 0, port)
}

func TestParseFromTo(t *testing.T) {
	// To
	displayInfo, uri, params := parseFromToContact(common.NetString("test <sip:test@10.0.2.15:5060>;tag=QvN921"))
	assert.Equal(t, common.NetString("test"), displayInfo)
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString("QvN921"), params["tag"])
	displayInfo, uri, params = parseFromToContact(common.NetString("test <sip:test@10.0.2.15:5060>"))
	assert.Equal(t, common.NetString("test"), displayInfo)
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString(nil), params["tag"])

	// From
	displayInfo, uri, params = parseFromToContact(common.NetString("\"PCMU/8000\"   <sip:sipp@10.0.2.15:5060>;tag=1"))
	assert.Equal(t, common.NetString("PCMU/8000"), displayInfo)
	assert.Equal(t, common.NetString("sip:sipp@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString("1"), params["tag"])
	displayInfo, uri, params = parseFromToContact(common.NetString("  \"Matthew Hodgson\" <sip:matthew@mxtelecom.com>;tag=5c7cdb68"))
	assert.Equal(t, common.NetString("Matthew Hodgson"), displayInfo)
	assert.Equal(t, common.NetString("sip:matthew@mxtelecom.com"), uri)
	assert.Equal(t, common.NetString("5c7cdb68"), params["tag"])
	displayInfo, uri, params = parseFromToContact(common.NetString("<sip:matthew@mxtelecom.com>;tag=5c7cdb68"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:matthew@mxtelecom.com"), uri)
	assert.Equal(t, common.NetString("5c7cdb68"), params["tag"])
	displayInfo, uri, params = parseFromToContact(common.NetString("<sip:matthew@mxtelecom.com>"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:matthew@mxtelecom.com"), uri)
	assert.Equal(t, common.NetString(nil), params["tag"])

	// Contact
	displayInfo, uri, _ = parseFromToContact(common.NetString("  <sip:test@10.0.2.15:5060;transport=udp>"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060;transport=udp"), uri)
	displayInfo, uri, params = parseFromToContact(common.NetString("<sip:voi18062@192.168.1.2:5060;line=aca6b97ca3f5e51a>;expires=1200;q=0.500"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:voi18062@192.168.1.2:5060;line=aca6b97ca3f5e51a"), uri)
	assert.Equal(t, common.NetString("1200"), params["expires"])
	assert.Equal(t, common.NetString("0.500"), params["q"])
	displayInfo, uri, params = parseFromToContact(common.NetString(" \"Mr. Watson\" <sip:watson@worcester.bell-telephone.com>;q=0.7; expires=3600"))
	assert.Equal(t, common.NetString("Mr. Watson"), displayInfo)
	assert.Equal(t, common.NetString("sip:watson@worcester.bell-telephone.com"), uri)
	assert.Equal(t, common.NetString("3600"), params["expires"])
	assert.Equal(t, common.NetString("0.7"), params["q"])
	displayInfo, uri, params = parseFromToContact(common.NetString(" \"Mr. Watson\" <mailto:watson@bell-telephone.com> ;q=0.1"))
	assert.Equal(t, common.NetString("Mr. Watson"), displayInfo)
	assert.Equal(t, common.NetString("mailto:watson@bell-telephone.com"), uri)
	assert.Equal(t, common.NetString("0.1"), params["q"])

	// url is not bounded by <...>
	displayInfo, uri, params = parseFromToContact(common.NetString("   sip:test@10.0.2.15:5060;transport=udp"))
	assert.Equal(t, common.NetString(nil), displayInfo)
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), uri)
	assert.Equal(t, common.NetString("udp"), params["transport"])
}

func TestParseUDP(t *testing.T) {
	gotEvent := new(beat.Event)
	reporter := func(evt beat.Event) {
		gotEvent = &evt
	}
	const data = "INVITE sip:test@10.0.2.15:5060 SIP/2.0\r\nVia: SIP/2.0/UDP 10.0.2.20:5060;branch=z9hG4bK-2187-1-0\r\nFrom: \"DVI4/8000\" <sip:sipp@10.0.2.20:5060>;tag=1\r\nTo: test <sip:test@10.0.2.15:5060>\r\nCall-ID: 1-2187@10.0.2.20\r\nCSeq: 1 INVITE\r\nContact: sip:sipp@10.0.2.20:5060\r\nMax-Forwards: 70\r\nContent-Type: application/sdp\r\nContent-Length:   123\r\n\r\nv=0\r\no=- 42 42 IN IP4 10.0.2.20\r\ns=-\r\nc=IN IP4 10.0.2.20\r\nt=0 0\r\nm=audio 6000 RTP/AVP 5\r\na=rtpmap:5 DVI4/8000\r\na=recvonly\r\n"
	p, _ := New(true, reporter, procs.ProcessesWatcher{}, nil)
	plugin := p.(*plugin)
	plugin.ParseUDP(&protos.Packet{
		Ts:      time.Now(),
		Tuple:   common.IPPortTuple{},
		Payload: []byte(data),
	})
	fields := *gotEvent

	assert.Equal(t, common.NetString("1-2187@10.0.2.20"), getVal(fields, "sip.call_id"))
	assert.Equal(t, common.NetString("test"), getVal(fields, "sip.contact.display_info"))
	assert.Equal(t, common.NetString("10.0.2.15"), getVal(fields, "sip.contact.uri.host"))
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), getVal(fields, "sip.contact.uri.original"))
	assert.Equal(t, 5060, getVal(fields, "sip.contact.uri.port"))
	assert.Equal(t, common.NetString("sip"), getVal(fields, "sip.contact.uri.scheme"))
	assert.Equal(t, common.NetString("test"), getVal(fields, "sip.contact.uri.username"))
	assert.Equal(t, 123, getVal(fields, "sip.content_length"))
	assert.Equal(t, common.NetString("application/sdp"), getVal(fields, "sip.content_type"))
	assert.Equal(t, 1, getVal(fields, "sip.cseq.code"))
	assert.Equal(t, common.NetString("INVITE"), getVal(fields, "sip.cseq.method"))
	assert.Equal(t, common.NetString("DVI4/8000"), getVal(fields, "sip.from.display_info"))
	assert.Equal(t, common.NetString("1"), getVal(fields, "sip.from.tag"))
	assert.Equal(t, common.NetString("10.0.2.20"), getVal(fields, "sip.from.uri.host"))
	assert.Equal(t, common.NetString("sip:sipp@10.0.2.20:5060"), getVal(fields, "sip.from.uri.original"))
	assert.Equal(t, 5060, getVal(fields, "sip.from.uri.port"))
	assert.Equal(t, common.NetString("sip"), getVal(fields, "sip.from.uri.scheme"))
	assert.Equal(t, common.NetString("sipp"), getVal(fields, "sip.from.uri.username"))
	assert.Equal(t, 70, getVal(fields, "sip.max_forwards"))
	assert.Equal(t, common.NetString("INVITE"), getVal(fields, "sip.method"))
	assert.Equal(t, common.NetString("10.0.2.20"), getVal(fields, "sip.sdp.connection.address"))
	assert.Equal(t, common.NetString("IN IP4 10.0.2.20"), getVal(fields, "sip.sdp.connection.info"))
	assert.Equal(t, common.NetString("10.0.2.20"), getVal(fields, "sip.sdp.owner.ip"))
	assert.Equal(t, common.NetString("42"), getVal(fields, "sip.sdp.owner.session_id"))
	assert.Equal(t, common.NetString("42"), getVal(fields, "sip.sdp.owner.version"))
	assert.Equal(t, nil, getVal(fields, "sip.sdp.owner.username"))
	assert.Equal(t, nil, getVal(fields, "sip.sdp.session.name"))
	assert.Equal(t, "0", getVal(fields, "sip.sdp.version"))
	assert.Equal(t, common.NetString("test"), getVal(fields, "sip.to.display_info"))
	assert.Equal(t, nil, getVal(fields, "sip.to.tag"))
	assert.Equal(t, common.NetString("10.0.2.15"), getVal(fields, "sip.to.uri.host"))
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), getVal(fields, "sip.to.uri.original"))
	assert.Equal(t, 5060, getVal(fields, "sip.to.uri.port"))
	assert.Equal(t, common.NetString("sip"), getVal(fields, "sip.to.uri.scheme"))
	assert.Equal(t, common.NetString("test"), getVal(fields, "sip.to.uri.username"))
	assert.Equal(t, "request", getVal(fields, "sip.type"))
	assert.Equal(t, common.NetString("10.0.2.15"), getVal(fields, "sip.uri.host"))
	assert.Equal(t, common.NetString("sip:test@10.0.2.15:5060"), getVal(fields, "sip.uri.original"))
	assert.Equal(t, 5060, getVal(fields, "sip.uri.port"))
	assert.Equal(t, common.NetString("sip"), getVal(fields, "sip.uri.scheme"))
	assert.Equal(t, common.NetString("test"), getVal(fields, "sip.uri.username"))
	assert.Equal(t, "2.0", getVal(fields, "sip.version"))
	assert.EqualValues(t, []common.NetString{common.NetString("SIP/2.0/UDP 10.0.2.20:5060;branch=z9hG4bK-2187-1-0")}, getVal(fields, "sip.via.original"))
}

func getVal(f beat.Event, k string) interface{} {
	v, _ := f.GetValue(k)
	return v
}
