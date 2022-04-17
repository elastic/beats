// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v6

import (
	"bytes"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
)

func TestNetflowProtocol_New(t *testing.T) {
	proto := New(config.Defaults())

	assert.Nil(t, proto.Start())
	assert.Equal(t, uint16(6), proto.Version())
	assert.Nil(t, proto.Stop())
}

func TestNetflowProtocol_OnPacket(t *testing.T) {
	proto := New(config.Defaults())

	rawS := "00060002000000015bf68d8b35fcb9780000000000000000" +
		"acd910e5c0a8017b00000000000000000000000e00002cfa" +
		"ffe8086cffe80f6201bbd711001806000000000000000000" +
		"00000000" + // extra padding, only difference with v5
		"c0a8017bacd910e500000000000000000000000700000c5b" +
		"ffe8086cffe80f62d71101bb001806000000000000000000" +
		"00000000" // extra padding, only difference with v5

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
			},
			Exporter: record.Map{
				"address":          "127.0.0.1:59707",
				"engineId":         uint64(0),
				"engineType":       uint64(0),
				"samplingInterval": uint64(0),
				"timestamp":        captureTime,
				"uptimeMillis":     uint64(1),
				"version":          uint64(6),
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
			},
			Exporter: record.Map{
				"address":          "127.0.0.1:59707",
				"engineId":         uint64(0),
				"engineType":       uint64(0),
				"samplingInterval": uint64(0),
				"timestamp":        captureTime,
				"uptimeMillis":     uint64(1),
				"version":          uint64(6),
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
	template.ValidateTemplate(t, &templateV6)
}
