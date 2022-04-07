// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v7

import (
	"bytes"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/test"
)

func TestNetflowProtocol_New(t *testing.T) {
	proto := New(config.Defaults())

	assert.Nil(t, proto.Start())
	assert.Equal(t, uint16(7), proto.Version())
	assert.Nil(t, proto.Stop())
}

func TestNetflowProtocol_OnPacket(t *testing.T) {
	proto := New(config.Defaults())

	rawS := "00070002000000015bf68d8b35fcb9780000000000000000" +
		"acd910e5c0a8017b00000000000000000000000e00002cfa" +
		"ffe8086cffe80f6201bbd711001806000000000000004411" +
		"ffffffff" + // extra fields
		"c0a8017bacd910e500000000000000000000000700000c5b" +
		"ffe8086cffe80f62d71101bb001806000000000000003322" +
		"fffefdfc" // extra fields

	captureTime, err := time.Parse(time.RFC3339Nano, "2018-11-22T11:05:47.905755Z")
	captureTime = captureTime.UTC()
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	expected := []record.Record{
		{
			Type:      record.Flow,
			Timestamp: captureTime,
			Fields: record.Map{
				"bgpDestinationAsNumber":      uint64(0),
				"bgpSourceAsNumber":           uint64(0),
				"destinationIPv4Address":      net.ParseIP("192.168.1.123").To4(),
				"destinationIPv4PrefixLength": uint64(0),
				"destinationTransportPort":    uint64(55057),
				"egressInterface":             uint64(0),
				"flowEndSysUpTime":            uint64(4293398370),
				"flowStartSysUpTime":          uint64(4293396588),
				"ingressInterface":            uint64(0),
				"ipClassOfService":            uint64(0),
				"ipNextHopIPv4Address":        net.ParseIP("0.0.0.0").To4(),
				"octetDeltaCount":             uint64(11514),
				"packetDeltaCount":            uint64(14),
				"protocolIdentifier":          uint64(6),
				"sourceIPv4Address":           net.ParseIP("172.217.16.229").To4(),
				"sourceIPv4PrefixLength":      uint64(0),
				"sourceTransportPort":         uint64(443),
				"tcpControlBits":              uint64(24),
				"flagsAndSamplerId":           uint64(0x4411),
				"ipv4RouterSc":                net.ParseIP("255.255.255.255").To4(),
			},
			Exporter: record.Map{
				"address":      "127.0.0.1:59707",
				"timestamp":    captureTime,
				"uptimeMillis": uint64(1),
				"version":      uint64(7),
			},
		}, {
			Type:      record.Flow,
			Timestamp: captureTime,
			Fields: record.Map{
				"bgpDestinationAsNumber":      uint64(0),
				"bgpSourceAsNumber":           uint64(0),
				"destinationIPv4Address":      net.ParseIP("172.217.16.229").To4(),
				"destinationIPv4PrefixLength": uint64(0),
				"destinationTransportPort":    uint64(443),
				"egressInterface":             uint64(0),
				"flowEndSysUpTime":            uint64(4293398370),
				"flowStartSysUpTime":          uint64(4293396588),
				"ingressInterface":            uint64(0),
				"ipClassOfService":            uint64(0),
				"ipNextHopIPv4Address":        net.ParseIP("0.0.0.0").To4(),
				"octetDeltaCount":             uint64(3163),
				"packetDeltaCount":            uint64(7),
				"protocolIdentifier":          uint64(6),
				"sourceIPv4Address":           net.ParseIP("192.168.1.123").To4(),
				"sourceIPv4PrefixLength":      uint64(0),
				"sourceTransportPort":         uint64(55057),
				"tcpControlBits":              uint64(24),
				"flagsAndSamplerId":           uint64(0x3322),
				"ipv4RouterSc":                net.ParseIP("255.254.253.252").To4(),
			},
			Exporter: record.Map{
				"address":      "127.0.0.1:59707",
				"timestamp":    captureTime,
				"uptimeMillis": uint64(1),
				"version":      uint64(7),
			},
		},
	}
	raw, err := hex.DecodeString(rawS)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	flows, err := proto.OnPacket(bytes.NewBuffer(raw), test.MakeAddress(t, "127.0.0.1:59707"))
	assert.NoError(t, err)
	assert.Len(t, flows, len(expected))
	assert.Equal(t, expected, flows)
}

func TestNetflowProtocol_BadPacket(t *testing.T) {
	proto := New(config.Defaults())

	rawS := "00060002000000015bf689f605"
	raw, err := hex.DecodeString(rawS)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	flows, err := proto.OnPacket(bytes.NewBuffer(raw), test.MakeAddress(t, "127.0.0.1:59707"))
	assert.Error(t, err)
	assert.Len(t, flows, 0)
}

func TestTemplate(t *testing.T) {
	template.ValidateTemplate(t, &v7template)
}
