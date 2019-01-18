// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v5

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/config"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/protocol"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/v1"
)

const (
	ProtocolName        = "v5"
	ProtocolID   uint16 = 5
	LogPrefix           = "[netflow-v5] "
)

var templateV5 = template.Template{
	ID: 0,
	Fields: []template.FieldTemplate{
		{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
		{Length: 4, Info: &fields.Field{Name: "destinationIPv4Address", Decoder: fields.Ipv4Address}},
		{Length: 4, Info: &fields.Field{Name: "ipNextHopIPv4Address", Decoder: fields.Ipv4Address}},
		{Length: 2, Info: &fields.Field{Name: "ingressInterface", Decoder: fields.Unsigned32}},
		{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
		{Length: 4, Info: &fields.Field{Name: "packetDeltaCount", Decoder: fields.Unsigned64}},
		{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
		{Length: 4, Info: &fields.Field{Name: "flowStartSysUpTime", Decoder: fields.Unsigned32}},
		{Length: 4, Info: &fields.Field{Name: "flowEndSysUpTime", Decoder: fields.Unsigned32}},
		{Length: 2, Info: &fields.Field{Name: "sourceTransportPort", Decoder: fields.Unsigned16}},
		{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
		{Length: 1}, // Padding
		{Length: 1, Info: &fields.Field{Name: "tcpControlBits", Decoder: fields.Unsigned16}},
		{Length: 1, Info: &fields.Field{Name: "protocolIdentifier", Decoder: fields.Unsigned8}},
		{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
		{Length: 2, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
		{Length: 2, Info: &fields.Field{Name: "bgpDestinationAsNumber", Decoder: fields.Unsigned32}},
		{Length: 1, Info: &fields.Field{Name: "sourceIPv4PrefixLength", Decoder: fields.Unsigned8}},
		{Length: 1, Info: &fields.Field{Name: "destinationIPv4PrefixLength", Decoder: fields.Unsigned8}},
		{Length: 2}, // Padding
	},
	Length: 48,
}

func init() {
	protocol.Registry.Register(ProtocolName, New)
}

func New(config config.Config) protocol.Protocol {
	return v1.NewProtocol(ProtocolID, &templateV5, ReadV5Header, log.New(config.LogOutput(), LogPrefix, 0))
}

type PacketHeader struct {
	Version          uint16
	Count            uint16
	SysUptime        uint32    // 32 bit milliseconds
	Timestamp        time.Time // 32 bit seconds + 32 bit nanoseconds
	FlowSequence     uint32
	EngineType       uint8
	EngineID         uint8
	SamplingInterval uint16
}

func ReadPacketHeader(buf *bytes.Buffer) (header PacketHeader, err error) {
	var arr [24]byte
	if n, err := buf.Read(arr[:]); err != nil || n != len(arr) {
		return header, io.EOF
	}
	timestamp := binary.BigEndian.Uint64(arr[8:16])
	header = PacketHeader{
		Version:          binary.BigEndian.Uint16(arr[:2]),
		Count:            binary.BigEndian.Uint16(arr[2:4]),
		SysUptime:        binary.BigEndian.Uint32(arr[4:8]),
		Timestamp:        time.Unix(int64(timestamp>>32), int64(timestamp&(1<<32-1))).UTC(),
		FlowSequence:     binary.BigEndian.Uint32(arr[16:20]),
		EngineType:       arr[20],
		EngineID:         arr[21],
		SamplingInterval: binary.BigEndian.Uint16(arr[22:]),
	}
	return header, nil
}

func ReadV5Header(buf *bytes.Buffer, source net.Addr) (count int, ts time.Time, metadata record.Map, err error) {
	header, err := ReadPacketHeader(buf)
	if err != nil {
		return count, ts, metadata, err
	}
	count = int(header.Count)
	metadata = record.Map{
		"version":          uint64(header.Version),
		"timestamp":        header.Timestamp,
		"uptimeMillis":     uint64(header.SysUptime),
		"address":          source.String(),
		"engineType":       uint64(header.EngineType),
		"engineId":         uint64(header.EngineID),
		"samplingInterval": uint64(header.SamplingInterval),
	}
	return count, header.Timestamp, metadata, nil
}
