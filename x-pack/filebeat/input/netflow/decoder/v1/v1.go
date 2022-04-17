// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v1

import (
	"bytes"
	"encoding/binary"
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
	ProtocolName        = "v1"
	LogPrefix           = "[netflow-v1] "
	ProtocolID   uint16 = 1
)

var templateV1 = template.Template{
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
		{Length: 2}, // Padding
		{Length: 1, Info: &fields.Field{Name: "protocolIdentifier", Decoder: fields.Unsigned8}},
		{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
		{Length: 1, Info: &fields.Field{Name: "tcpControlBits", Decoder: fields.Unsigned16}},
		{Length: 7}, // Padding
	},
	Length: 48,
}

type ReadHeaderFn func(*bytes.Buffer, net.Addr) (int, time.Time, record.Map, error)

type NetflowProtocol struct {
	logger       *log.Logger
	flowTemplate *template.Template
	version      uint16
	readHeader   ReadHeaderFn
}

func init() {
	protocol.Registry.Register(ProtocolName, New)
}

func New(config config.Config) protocol.Protocol {
	return NewProtocol(ProtocolID, &templateV1, readV1Header, log.New(config.LogOutput(), LogPrefix, 0))
}

func NewProtocol(version uint16, template *template.Template, readHeader ReadHeaderFn, logger *log.Logger) protocol.Protocol {
	return &NetflowProtocol{
		logger:       logger,
		flowTemplate: template,
		version:      version,
		readHeader:   readHeader,
	}
}

func (p *NetflowProtocol) Version() uint16 {
	return p.version
}

func (NetflowProtocol) Start() error {
	return nil
}

func (NetflowProtocol) Stop() error {
	return nil
}

func (p *NetflowProtocol) OnPacket(buf *bytes.Buffer, source net.Addr) (flows []record.Record, err error) {
	numFlows, timestamp, metadata, err := p.readHeader(buf, source)
	if err != nil {
		p.logger.Printf("Failed parsing packet: %v", err)
		return nil, errors.Wrap(err, "error reading netflow header")
	}
	flows, err = p.flowTemplate.Apply(buf, numFlows)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing flows")
	}
	for i := range flows {
		flows[i].Exporter = metadata
		flows[i].Timestamp = timestamp
	}
	return flows, nil
}

type PacketHeader struct {
	Version   uint16
	Count     uint16
	SysUptime uint32    // 32 bit milliseconds
	Timestamp time.Time // 32 bit seconds + 32 bit nanoseconds
}

func ReadPacketHeader(buf *bytes.Buffer) (header PacketHeader, err error) {
	var arr [16]byte
	if n, err := buf.Read(arr[:]); err != nil || n != len(arr) {
		return header, io.EOF
	}
	timestamp := binary.BigEndian.Uint64(arr[8:16])
	header = PacketHeader{
		Version:   binary.BigEndian.Uint16(arr[:2]),
		Count:     binary.BigEndian.Uint16(arr[2:4]),
		SysUptime: binary.BigEndian.Uint32(arr[4:8]),
		Timestamp: time.Unix(int64(timestamp>>32), int64(timestamp&(1<<32-1))).UTC(),
	}
	return header, nil
}

func readV1Header(buf *bytes.Buffer, source net.Addr) (count int, ts time.Time, metadata record.Map, err error) {
	header, err := ReadPacketHeader(buf)
	if err != nil {
		return count, ts, metadata, err
	}
	count = int(header.Count)
	metadata = record.Map{
		"version":      uint64(header.Version),
		"timestamp":    header.Timestamp,
		"uptimeMillis": uint64(header.SysUptime),
		"address":      source.String(),
	}
	return count, header.Timestamp, metadata, nil
}
