// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v1

import (
	"bytes"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	template2 "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
)

func TestNetflowProtocol_New(t *testing.T) {
	proto := New(config.Defaults())

	assert.Nil(t, proto.Start())
	assert.Equal(t, uint16(1), proto.Version())
	assert.Nil(t, proto.Stop())
}

func TestNetflowProtocol_OnPacket(t *testing.T) {
	proto := New(config.Defaults())

	rawS := "00010002000000015bf689f605946fb0" +
		"acd910e5c0a8017b00000000000000000000000e00002cfa" +
		"fff609a0fff6109601bbd711000006001800000000000000" +
		"c0a8017bacd910e500000000000000000000000700000c5b" +
		"fff609a0fff61096d71101bb000006001800000000000000"

	captureTime, err := time.Parse(time.RFC3339Nano, "2018-11-22T10:50:30.093614Z")
	captureTime = captureTime.UTC()
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	expected := []record.Record{
		{
			Type:      record.Flow,
			Timestamp: captureTime,
			Fields: record.Map{
				"destinationIPv4Address":   net.ParseIP("192.168.1.123").To4(),
				"destinationTransportPort": uint64(55057),
				"egressInterface":          uint64(0),
				"flowEndSysUpTime":         uint64(4294316182),
				"flowStartSysUpTime":       uint64(4294314400),
				"ingressInterface":         uint64(0),
				"ipClassOfService":         uint64(0),
				"ipNextHopIPv4Address":     net.ParseIP("0.0.0.0").To4(),
				"octetDeltaCount":          uint64(11514),
				"packetDeltaCount":         uint64(14),
				"protocolIdentifier":       uint64(6),
				"sourceIPv4Address":        net.ParseIP("172.217.16.229").To4(),
				"sourceTransportPort":      uint64(443),
				"tcpControlBits":           uint64(24),
			},
			Exporter: record.Map{
				"address":      "127.0.0.1:59707",
				"timestamp":    captureTime,
				"uptimeMillis": uint64(1),
				"version":      uint64(1),
			},
		}, {
			Type:      record.Flow,
			Timestamp: captureTime,
			Fields: record.Map{
				"destinationIPv4Address":   net.ParseIP("172.217.16.229").To4(),
				"destinationTransportPort": uint64(443),
				"egressInterface":          uint64(0),
				"flowEndSysUpTime":         uint64(4294316182),
				"flowStartSysUpTime":       uint64(4294314400),
				"ingressInterface":         uint64(0),
				"ipClassOfService":         uint64(0),
				"ipNextHopIPv4Address":     net.ParseIP("0.0.0.0").To4(),
				"octetDeltaCount":          uint64(3163),
				"packetDeltaCount":         uint64(7),
				"protocolIdentifier":       uint64(6),
				"sourceIPv4Address":        net.ParseIP("192.168.1.123").To4(),
				"sourceTransportPort":      uint64(55057),
				"tcpControlBits":           uint64(24),
			},
			Exporter: record.Map{
				"address":      "127.0.0.1:59707",
				"timestamp":    captureTime,
				"uptimeMillis": uint64(1),
				"version":      uint64(1),
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

	rawS := "00010002000000015bf689f605"
	raw, err := hex.DecodeString(rawS)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	flows, err := proto.OnPacket(bytes.NewBuffer(raw), test.MakeAddress(t, "127.0.0.1:59707"))
	assert.Error(t, err)
	assert.Len(t, flows, 0)
}

func TestTemplate(t *testing.T) {
	template2.ValidateTemplate(t, &templateV1)
}
