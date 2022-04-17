// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v8

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	template2 "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
)

func TestTemplates(t *testing.T) {
	for code, template := range templates {
		if !template2.ValidateTemplate(t, template) {
			t.Fatal("Failed validating template for V8 record", code)
		}
	}
}

func TestNetflowProtocol_New(t *testing.T) {
	proto := New(config.Defaults())

	assert.Nil(t, proto.Start())
	assert.Equal(t, uint16(8), proto.Version())
	assert.Nil(t, proto.Stop())
}

func TestNetflowProtocol_BadPacket(t *testing.T) {
	proto := New(config.Defaults())

	rawS := "00080002000000015bf689f605"
	raw, err := hex.DecodeString(rawS)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	flows, err := proto.OnPacket(bytes.NewBuffer(raw), test.MakeAddress(t, "127.0.0.1:59707"))
	assert.Error(t, err)
	assert.Len(t, flows, 0)
}

func TestNetflowV8Protocol_OnPacket(t *testing.T) {
	proto := New(config.Defaults())
	address := test.MakeAddress(t, "127.0.0.1:11111")
	captureTime, err := time.Parse(time.RFC3339Nano, "2018-11-22T20:53:03.987654321Z")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name        string
		aggregation AggType
		packet      []uint16
		expected    record.Record
		empty       bool
		err         error
	}{
		{
			name:        "RouterAS",
			aggregation: RouterAS,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":         uint64(0x12345678),
					"packetDeltaCount":       uint64(0x9abcdef),
					"octetDeltaCount":        uint64(0x11223344),
					"flowStartSysUpTime":     uint64(0x55667788),
					"flowEndSysUpTime":       uint64(0x99aa99bb),
					"bgpSourceAsNumber":      uint64(0x1111),
					"bgpDestinationAsNumber": uint64(0x2222),
					"ingressInterface":       uint64(0x3333),
					"egressInterface":        uint64(0x4444),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(RouterAS),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "RouterProtoPort",
			aggregation: RouterProtoPort,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":           uint64(0x12345678),
					"packetDeltaCount":         uint64(0x9abcdef),
					"octetDeltaCount":          uint64(0x11223344),
					"flowStartSysUpTime":       uint64(0x55667788),
					"flowEndSysUpTime":         uint64(0x99aa99bb),
					"protocolIdentifier":       uint64(0x11),
					"sourceTransportPort":      uint64(0x3333),
					"destinationTransportPort": uint64(0x4444),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(RouterProtoPort),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "RouterDstPrefix",
			aggregation: RouterDstPrefix,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x0506, 0,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":         uint64(0x12345678),
					"packetDeltaCount":       uint64(0x09abcdef),
					"octetDeltaCount":        uint64(0x11223344),
					"flowStartSysUpTime":     uint64(0x55667788),
					"flowEndSysUpTime":       uint64(0x99aa99bb),
					"destinationIPv4Prefix":  net.ParseIP("17.17.34.34").To4(),
					"bgpDestinationAsNumber": uint64(0x4444),
					"egressInterface":        uint64(0x0506),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(RouterDstPrefix),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "RouterSrcPrefix",
			aggregation: RouterSrcPrefix,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x0506, 0,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":     uint64(0x12345678),
					"packetDeltaCount":   uint64(0x09abcdef),
					"octetDeltaCount":    uint64(0x11223344),
					"flowStartSysUpTime": uint64(0x55667788),
					"flowEndSysUpTime":   uint64(0x99aa99bb),
					"sourceIPv4Prefix":   net.ParseIP("17.17.34.34").To4(),
					"bgpSourceAsNumber":  uint64(0x4444),
					"ingressInterface":   uint64(0x0506),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(RouterSrcPrefix),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "RouterPrefix",
			aggregation: RouterPrefix,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0, 0,
				0x0506, 0x0708, 0x090a, 0x0b0c, 0x0d0e,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":         uint64(0x12345678),
					"packetDeltaCount":       uint64(0x9abcdef),
					"octetDeltaCount":        uint64(0x11223344),
					"flowStartSysUpTime":     uint64(0x55667788),
					"flowEndSysUpTime":       uint64(0x99aa99bb),
					"sourceIPv4Prefix":       net.ParseIP("17.17.34.34").To4(),
					"destinationIPv4Prefix":  net.ParseIP("51.51.68.68").To4(),
					"bgpSourceAsNumber":      uint64(0x0506),
					"bgpDestinationAsNumber": uint64(0x0708),
					"ingressInterface":       uint64(0x090a),
					"egressInterface":        uint64(0x0b0c),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(RouterPrefix),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "TosAS",
			aggregation: TosAS,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":         uint64(0x12345678),
					"packetDeltaCount":       uint64(0x09abcdef),
					"octetDeltaCount":        uint64(0x11223344),
					"flowStartSysUpTime":     uint64(0x55667788),
					"flowEndSysUpTime":       uint64(0x99aa99bb),
					"bgpSourceAsNumber":      uint64(0x1111),
					"bgpDestinationAsNumber": uint64(0x2222),
					"ingressInterface":       uint64(0x3333),
					"egressInterface":        uint64(0x4444),
					"ipClassOfService":       uint64(0x55),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(TosAS),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "TosProtoPort",
			aggregation: TosProtoPort,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":           uint64(0x12345678),
					"packetDeltaCount":         uint64(0x9abcdef),
					"octetDeltaCount":          uint64(0x11223344),
					"flowStartSysUpTime":       uint64(0x55667788),
					"flowEndSysUpTime":         uint64(0x99aa99bb),
					"protocolIdentifier":       uint64(0x11),
					"ipClassOfService":         uint64(0x11),
					"sourceTransportPort":      uint64(0x3333),
					"destinationTransportPort": uint64(0x4444),
					"ingressInterface":         uint64(0x5555),
					"egressInterface":          uint64(0x6666),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(TosProtoPort),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "PrePortProtocol",
			aggregation: PrePortProtocol,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
				0x7181, 0x91a1, 0xb1c1, 0xd1e1,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":              uint64(0x12345678),
					"packetDeltaCount":            uint64(0x9abcdef),
					"octetDeltaCount":             uint64(0x11223344),
					"flowStartSysUpTime":          uint64(0x55667788),
					"flowEndSysUpTime":            uint64(0x99aa99bb),
					"sourceIPv4Prefix":            net.ParseIP("17.17.34.34").To4(),
					"destinationIPv4Prefix":       net.ParseIP("51.51.68.68").To4(),
					"destinationIPv4PrefixLength": uint64(0x55),
					"sourceIPv4PrefixLength":      uint64(0x55),
					"ipClassOfService":            uint64(0x66),
					"protocolIdentifier":          uint64(0x66),
					"sourceTransportPort":         uint64(0x7181),
					"destinationTransportPort":    uint64(0x91a1),
					"ingressInterface":            uint64(0xb1c1),
					"egressInterface":             uint64(0xd1e1),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(PrePortProtocol),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "TosSrcPrefix",
			aggregation: TosSrcPrefix,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":         uint64(0x12345678),
					"packetDeltaCount":       uint64(0x9abcdef),
					"octetDeltaCount":        uint64(0x11223344),
					"flowStartSysUpTime":     uint64(0x55667788),
					"flowEndSysUpTime":       uint64(0x99aa99bb),
					"sourceIPv4Prefix":       net.ParseIP("17.17.34.34").To4(),
					"sourceIPv4PrefixLength": uint64(0x33),
					"ipClassOfService":       uint64(0x33),
					"bgpSourceAsNumber":      uint64(0x4444),
					"ingressInterface":       uint64(0x5555),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(TosSrcPrefix),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "TosDstPrefix",
			aggregation: TosDstPrefix,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":              uint64(0x12345678),
					"packetDeltaCount":            uint64(0x9abcdef),
					"octetDeltaCount":             uint64(0x11223344),
					"flowStartSysUpTime":          uint64(0x55667788),
					"flowEndSysUpTime":            uint64(0x99aa99bb),
					"destinationIPv4Prefix":       net.ParseIP("17.17.34.34").To4(),
					"destinationIPv4PrefixLength": uint64(0x33),
					"ipClassOfService":            uint64(0x33),
					"bgpDestinationAsNumber":      uint64(0x4444),
					"egressInterface":             uint64(0x5555),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(TosDstPrefix),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "TosPrefix",
			aggregation: TosPrefix,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
				0x7181, 0x91a1, 0xb1c1, 0xd1e1,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"deltaFlowCount":              uint64(0x12345678),
					"packetDeltaCount":            uint64(0x9abcdef),
					"octetDeltaCount":             uint64(0x11223344),
					"flowStartSysUpTime":          uint64(0x55667788),
					"flowEndSysUpTime":            uint64(0x99aa99bb),
					"sourceIPv4Prefix":            net.ParseIP("17.17.34.34").To4(),
					"destinationIPv4Prefix":       net.ParseIP("51.51.68.68").To4(),
					"destinationIPv4PrefixLength": uint64(0x55),
					"sourceIPv4PrefixLength":      uint64(0x55),
					"ipClassOfService":            uint64(0x66),
					"bgpSourceAsNumber":           uint64(0x7181),
					"bgpDestinationAsNumber":      uint64(0x91a1),
					"ingressInterface":            uint64(0xb1c1),
					"egressInterface":             uint64(0xd1e1),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(TosPrefix),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "DestOnly",
			aggregation: DestOnly,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"destinationIPv4Address":  net.ParseIP("18.52.86.120").To4(),
					"packetDeltaCount":        uint64(0x9abcdef),
					"octetDeltaCount":         uint64(0x11223344),
					"flowStartSysUpTime":      uint64(0x55667788),
					"flowEndSysUpTime":        uint64(0x99aa99bb),
					"egressInterface":         uint64(0x1111),
					"ipClassOfService":        uint64(0x22),
					"postIpClassOfService":    uint64(0x22),
					"droppedPacketDeltaCount": uint64(0x33334444),
					"ipv4RouterSc":            net.ParseIP("85.85.102.102").To4(),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(DestOnly),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "SrcDst",
			aggregation: SrcDst,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
				0x7181, 0x91a1, 0xb1c1, 0xd1e1,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"destinationIPv4Address":  net.ParseIP("18.52.86.120").To4(),
					"sourceIPv4Address":       net.ParseIP("9.171.205.239").To4(),
					"packetDeltaCount":        uint64(0x11223344),
					"octetDeltaCount":         uint64(0x55667788),
					"flowStartSysUpTime":      uint64(0x99aa99bb),
					"flowEndSysUpTime":        uint64(0x11112222),
					"egressInterface":         uint64(0x3333),
					"ingressInterface":        uint64(0x4444),
					"ipClassOfService":        uint64(0x55),
					"postIpClassOfService":    uint64(0x55),
					"droppedPacketDeltaCount": uint64(0x718191a1),
					"ipv4RouterSc":            net.ParseIP("177.193.209.225").To4(),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(SrcDst),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "FullFlow",
			aggregation: FullFlow,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
				0x7181, 0x91a1, 0xb1c1, 0xd1e1, 0x2f2e, 0x2d2c,
			},
			expected: record.Record{
				Type:      record.Flow,
				Timestamp: captureTime,
				Fields: record.Map{
					"destinationIPv4Address":   net.ParseIP("18.52.86.120").To4(),
					"sourceIPv4Address":        net.ParseIP("9.171.205.239").To4(),
					"destinationTransportPort": uint64(0x1122),
					"sourceTransportPort":      uint64(0x3344),
					"packetDeltaCount":         uint64(0x55667788),
					"octetDeltaCount":          uint64(0x99aa99bb),
					"flowStartSysUpTime":       uint64(0x11112222),
					"flowEndSysUpTime":         uint64(0x33334444),
					"egressInterface":          uint64(0x5555),
					"ingressInterface":         uint64(0x6666),
					"ipClassOfService":         uint64(0x71),
					"protocolIdentifier":       uint64(0x81),
					"postIpClassOfService":     uint64(0x91),
					"droppedPacketDeltaCount":  uint64(0xb1c1d1e1),
					"ipv4RouterSc":             net.ParseIP("47.46.45.44").To4(),
				},
				Exporter: record.Map{
					"version":            uint64(8),
					"timestamp":          captureTime,
					"uptimeMillis":       uint64(0x10002),
					"address":            address.String(),
					"engineType":         uint64(1),
					"engineId":           uint64(2),
					"aggregation":        uint64(FullFlow),
					"aggregationVersion": uint64(0),
				},
			},
		},
		{
			name:        "Unknown",
			aggregation: 0xff,
			packet: []uint16{
				// Header
				8, 1, 1, 2, 23543, 5935, 15070, 26801, 0x1234, 0x5678, 258, 0, 0, 0,
				// Flow record
				0x1234, 0x5678, 0x09ab, 0xcdef, 0x1122, 0x3344, 0x5566, 0x7788,
				0x99aa, 0x99bb, 0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666,
				0x7181, 0x91a1, 0xb1c1, 0xd1e1, 0x2f2e, 0x2d2c,
			},
			empty: true,
			err:   errors.New("unsupported V8 aggregation: 255"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			raw := test.MakePacket(testCase.packet)
			raw.Bytes()[22] = uint8(testCase.aggregation)
			flow, err := proto.OnPacket(raw, address)
			if err == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, testCase.err, err)
			}
			if !testCase.empty {
				if !assert.Len(t, flow, 1) {
					return
				}
				test.AssertRecordsEqual(t, testCase.expected, flow[0])
			} else {
				assert.Empty(t, flow)
			}
		})
	}
}
