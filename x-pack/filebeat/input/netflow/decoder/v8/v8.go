// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v8

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
)

const (
	ProtocolName        = "v8"
	LogPrefix           = "[netflow-v8] "
	ProtocolID   uint16 = 8
)

// AggType is an enumeration type for Netflow V8 aggregations.
// See https://www.cisco.com/c/en/us/td/docs/net_mgmt/netflow_collection_engine/3-6/user/guide/format.html
type AggType uint8

const (
	RouterAS AggType = iota + 1
	RouterProtoPort
	RouterSrcPrefix
	RouterDstPrefix
	RouterPrefix
	DestOnly
	SrcDst
	FullFlow
	TosAS
	TosProtoPort
	TosSrcPrefix
	TosDstPrefix
	TosPrefix
	PrePortProtocol
)

var templates = map[AggType]*template.Template{
	RouterAS: {
		Fields: []template.FieldTemplate{
			//  observedFlowTotalCount
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
		},
		Length: 28,
	},
	RouterProtoPort: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 1, Info: &fields.Field{Name: "protocolIdentifier", Decoder: fields.Unsigned8}},
			{Length: 3},
			{Length: 2, Info: &fields.Field{Name: "sourceTransportPort", Decoder: fields.Unsigned16}},
			{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
		},
		Length: 28,
	},
	RouterDstPrefix: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 2},
			{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
			{Length: 2},
		},
		Length: 32,
	},
	RouterSrcPrefix: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 2},
			{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2},
		},
		Length: 32,
	},
	RouterPrefix: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 4},
			{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
		},
		Length: 40,
	},
	TosAS: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 3},
		},
		Length: 32,
	},
	TosProtoPort: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 1, Info: &fields.Field{Name: "protocolIdentifier", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 2},
			{Length: 2, Info: &fields.Field{Name: "sourceTransportPort", Decoder: fields.Unsigned16}},
			{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
		},
		Length: 32,
	},
	PrePortProtocol: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Prefix", Decoder: fields.Ipv4Address}},
			// Warning: according to CISCO docs, this is reversed (dest, src)
			{Length: 1, Info: &fields.Field{Name: "destinationIPv4PrefixLength", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "sourceIPv4PrefixLength", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "protocolIdentifier", Decoder: fields.Unsigned8}},
			{Length: 2, Info: &fields.Field{Name: "sourceTransportPort", Decoder: fields.Unsigned16}},
			{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
		},
		Length: 40,
	},
	TosSrcPrefix: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 1, Info: &fields.Field{Name: "sourceIPv4PrefixLength", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2},
		},
		Length: 32,
	},
	TosDstPrefix: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 1, Info: &fields.Field{Name: "destinationIPv4PrefixLength", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
			{Length: 2},
		},
		Length: 32,
	},
	TosPrefix: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "deltaFlowCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Prefix", Decoder: fields.Ipv4Address}},
			{Length: 1, Info: &fields.Field{Name: "destinationIPv4PrefixLength", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "sourceIPv4PrefixLength", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 1},
			{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
		},
		Length: 40,
	},
	DestOnly: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Address", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			// Warning: This is documented as "marked_tos: Type of Service of the packets that exceeded the contract"
			//          but I can't find a V9 field for it.
			{Length: 1, Info: &fields.Field{Name: "postIpClassOfService", Decoder: fields.Unsigned8}},
			// Warning: This is documented as "extraPkts: Packets that exceeded the contract"
			//          but I can't find a V9 field for it.
			{Length: 4, Info: &fields.Field{Name: "droppedPacketDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "ipv4RouterSc", Decoder: fields.Ipv4Address}},
		},
		Length: 32,
	},
	SrcDst: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Address", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			// Warning: This is documented as "marked_tos: Type of Service of the packets that exceeded the contract"
			//          but I can't find a V9 field for it.
			{Length: 1, Info: &fields.Field{Name: "postIpClassOfService", Decoder: fields.Unsigned8}},
			{Length: 2}, // Padding
			// Warning: This is documented as "extraPkts: Packets that exceeded the contract"
			//          but I can't find a V9 field for it.
			{Length: 4, Info: &fields.Field{Name: "droppedPacketDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "ipv4RouterSc", Decoder: fields.Ipv4Address}},
		},
		Length: 40,
	},
	FullFlow: {
		Fields: []template.FieldTemplate{
			{Length: 4, Info: &fields.Field{Name: "destinationIPv4Address", Decoder: fields.Ipv4Address}},
			{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
			{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
			{Length: 2, Info: &fields.Field{Name: "sourceTransportPort", Decoder: fields.Unsigned16}},
			{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
			{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},

			{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
			{Length: 1, Info: &fields.Field{Name: "protocolIdentifier", Decoder: fields.Unsigned8}},
			// Warning: This is documented as "marked_tos: Type of Service of the packets that exceeded the contract"
			//          but I can't find a V9 field for it.
			{Length: 1, Info: &fields.Field{Name: "postIpClassOfService", Decoder: fields.Unsigned8}},
			{Length: 1}, // Padding
			// Warning: This is documented as "extraPkts: Packets that exceeded the contract"
			//          but I can't find a V9 field for it.
			{Length: 4, Info: &fields.Field{Name: "droppedPacketDeltaCount", Decoder: fields.Unsigned64}},
			{Length: 4, Info: &fields.Field{Name: "ipv4RouterSc", Decoder: fields.Ipv4Address}},
		},
		Length: 44,
	},
}

type NetflowV8Protocol struct {
	logger *log.Logger
}

func init() {
	protocol.Registry.Register(ProtocolName, New)
}

func New(config config.Config) protocol.Protocol {
	return &NetflowV8Protocol{
		logger: log.New(config.LogOutput(), LogPrefix, 0),
	}
}

func (NetflowV8Protocol) Version() uint16 {
	return ProtocolID
}

func (p *NetflowV8Protocol) OnPacket(buf *bytes.Buffer, source net.Addr) (flows []record.Record, err error) {
	header, err := ReadPacketHeader(buf)
	if err != nil {
		p.logger.Printf("Failed parsing packet: %v", err)
		return nil, errors.Wrap(err, "error reading V8 header")
	}
	template, found := templates[header.Aggregation]
	if !found {
		p.logger.Printf("Packet from %s uses an unknown V8 aggregation: %d", source, header.Aggregation)
		return nil, fmt.Errorf("unsupported V8 aggregation: %d", header.Aggregation)
	}
	metadata := header.GetMetadata(source)
	flows, err = template.Apply(buf, int(header.Count))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to decode V8 flows of type %d", header.Aggregation)
	}
	for i := range flows {
		flows[i].Exporter = metadata
		flows[i].Timestamp = header.Timestamp
	}
	return flows, nil
}

func (NetflowV8Protocol) Start() error {
	return nil
}

func (NetflowV8Protocol) Stop() error {
	return nil
}

type PacketHeader struct {
	Version      uint16
	Count        uint16
	SysUptime    uint32    // 32 bit milliseconds
	Timestamp    time.Time // 32 bit seconds + 32 bit nanoseconds
	FlowSequence uint32
	EngineType   uint8
	EngineID     uint8
	Aggregation  AggType
	AggVersion   uint8
	Reserved     uint32
}

func ReadPacketHeader(buf *bytes.Buffer) (header PacketHeader, err error) {
	var arr [28]byte
	if n, err := buf.Read(arr[:]); err != nil || n != len(arr) {
		if err == nil {
			err = io.EOF
		}
		return header, err
	}
	timestamp := binary.BigEndian.Uint64(arr[8:16])
	header = PacketHeader{
		Version:      binary.BigEndian.Uint16(arr[:2]),
		Count:        binary.BigEndian.Uint16(arr[2:4]),
		SysUptime:    binary.BigEndian.Uint32(arr[4:8]),
		Timestamp:    time.Unix(int64(timestamp>>32), int64(timestamp&(1<<32-1))).UTC(),
		FlowSequence: binary.BigEndian.Uint32(arr[16:20]),
		EngineType:   arr[20],
		EngineID:     arr[21],
		Aggregation:  AggType(arr[22]),
		AggVersion:   arr[23],
	}
	return header, nil
}

func (header PacketHeader) GetMetadata(source net.Addr) record.Map {
	return record.Map{
		"version":            uint64(header.Version),
		"timestamp":          header.Timestamp,
		"uptimeMillis":       uint64(header.SysUptime),
		"address":            source.String(),
		"engineType":         uint64(header.EngineType),
		"engineId":           uint64(header.EngineID),
		"aggregation":        uint64(header.Aggregation),
		"aggregationVersion": uint64(header.AggVersion),
	}
}
