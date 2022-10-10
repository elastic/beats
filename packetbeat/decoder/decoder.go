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

package decoder

import (
	"fmt"
	"sort"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/icmp"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
	"github.com/elastic/beats/v7/packetbeat/protos/udp"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Decoder struct {
	decoders         map[gopacket.LayerType]gopacket.DecodingLayer
	linkLayerDecoder gopacket.DecodingLayer
	linkLayerType    gopacket.LayerType

	sll       layers.LinuxSLL
	lo        layers.Loopback
	eth       layers.Ethernet
	d1q       [2]layers.Dot1Q
	ip4       [2]layers.IPv4
	ip6       [2]layers.IPv6
	icmp4     layers.ICMPv4
	icmp6     layers.ICMPv6
	tcp       layers.TCP
	udp       layers.UDP
	truncated bool

	fragments []fragment

	stD1Q, stIP4, stIP6 multiLayer

	icmp4Proc icmp.ICMPv4Processor
	icmp6Proc icmp.ICMPv6Processor
	tcpProc   tcp.Processor
	udpProc   udp.Processor

	flows          *flows.Flows
	statPackets    *flows.Uint
	statBytes      *flows.Uint
	icmpV4TypeCode *flows.Uint
	icmpV6TypeCode *flows.Uint

	// hold current flow ID
	flowID              *flows.FlowID // buffer flowID among many calls
	flowIDBufferBacking [flows.SizeFlowIDMax]byte
}

// maxReconstruct is the maximum size that a collection of fragmented
// packets will be reconstructed to.
const maxReconstruct = 1e5

type fragment struct {
	id     uint16
	offset int
	data   []byte
}

const (
	netPacketsTotalCounter = "packets"
	netBytesTotalCounter   = "bytes"
	icmpV4TypeCodeValue    = "icmpV4TypeCode"
	icmpV6TypeCodeValue    = "icmpV6TypeCode"
)

// New creates and initializes a new packet decoder.
func New(f *flows.Flows, datalink layers.LinkType, icmp4 icmp.ICMPv4Processor, icmp6 icmp.ICMPv6Processor, tcp tcp.Processor, udp udp.Processor) (*Decoder, error) {
	d := Decoder{
		flows:     f,
		decoders:  make(map[gopacket.LayerType]gopacket.DecodingLayer),
		icmp4Proc: icmp4, icmp6Proc: icmp6, tcpProc: tcp, udpProc: udp,
	}
	d.stD1Q.init(&d.d1q[0], &d.d1q[1])
	d.stIP4.init(&d.ip4[0], &d.ip4[1])
	d.stIP6.init(&d.ip6[0], &d.ip6[1])

	if f != nil {
		var err error
		d.statPackets, err = f.NewUint(netPacketsTotalCounter)
		if err != nil {
			return nil, err
		}
		d.statBytes, err = f.NewUint(netBytesTotalCounter)
		if err != nil {
			return nil, err
		}
		d.icmpV4TypeCode, err = f.NewUint(icmpV4TypeCodeValue)
		if err != nil {
			return nil, err
		}
		d.icmpV6TypeCode, err = f.NewUint(icmpV6TypeCodeValue)
		if err != nil {
			return nil, err
		}

		d.flowID = &flows.FlowID{}
	}

	d.AddLayers([]gopacket.DecodingLayer{
		&d.sll,             // LinuxSLL
		&d.eth,             // Ethernet
		&d.lo,              // loopback on OS X
		&d.stD1Q,           // VLAN
		&d.stIP4, &d.stIP6, // IP
		&d.icmp4, &d.icmp6, // ICMP
		&d.tcp, &d.udp, // TCP/UDP
	})

	logp.Debug("decoder", "Layer type: %s", datalink)

	switch datalink {
	case layers.LinkTypeLinuxSLL:
		d.linkLayerDecoder = &d.sll
		d.linkLayerType = layers.LayerTypeLinuxSLL
	case layers.LinkTypeEthernet:
		d.linkLayerDecoder = &d.eth
		d.linkLayerType = layers.LayerTypeEthernet
	case layers.LinkTypeNull: // loopback on OSx
		d.linkLayerDecoder = &d.lo
		d.linkLayerType = layers.LayerTypeLoopback
	default:
		return nil, fmt.Errorf("unsupported link type: %s", datalink)
	}

	return &d, nil
}

func (d *Decoder) SetTruncated() {
	d.truncated = true
}

func (d *Decoder) AddLayer(layer gopacket.DecodingLayer) {
	for _, typ := range layer.CanDecode().LayerTypes() {
		d.decoders[typ] = layer
	}
}

func (d *Decoder) AddLayers(layers []gopacket.DecodingLayer) {
	for _, layer := range layers {
		d.AddLayer(layer)
	}
}

func (d *Decoder) OnPacket(data []byte, ci *gopacket.CaptureInfo) {
	defer logp.Recover("packet decoding failed")

	d.truncated = false

	current := d.linkLayerDecoder
	currentType := d.linkLayerType

	packet := protos.Packet{Ts: ci.Timestamp}

	logp.Debug("decoder", "decode packet data")

	if d.flowID != nil {
		d.flowID.Reset(d.flowIDBufferBacking[:0])

		// suppress flow stats snapshots while processing packet
		d.flows.Lock()
		defer d.flows.Unlock()
	}

	d.stD1Q.i = 0
	d.stIP4.i = 0
	d.stIP6.i = 0
	for len(data) != 0 {
		err := current.DecodeFromBytes(data, d)
		if err != nil {
			logp.Info("packet decode failed with: %v", err)
			break
		}

		nextType := current.NextLayerType()
		data = current.LayerPayload()
		if nextType == gopacket.LayerTypeFragment {
			ipv4, ok := ipv4Layer(current)
			if !ok {
				// This should never happen.
				logp.Warn("no IPv4 layer for fragment")
			} else {
				// FIXME: The current code assumes that the final packet in the
				// set of fragments arrives last. This is not guaranteed to happen.
				// An approach to dealing with this would be to keep a time-to-live
				// store keyed on the packet ID.

				const offsetMask = 1<<13 - 1 // https://datatracker.ietf.org/doc/html/rfc791#section-3.1
				more := ipv4.Flags&layers.IPv4MoreFragments != 0
				d.fragments = append(d.fragments, fragment{
					id:     ipv4.Id,
					offset: int(ipv4.FragOffset&offsetMask) * 8,
					data:   append(data[:0:0], data...), // Ensure that we are not aliasing data.
				})
				if more {
					return
				}
				sort.Slice(d.fragments, func(i, j int) bool {
					return d.fragments[i].offset < d.fragments[j].offset
				})
				id := d.fragments[0].id
				data = d.fragments[0].data
				for _, f := range d.fragments[1:] {
					if f.id != id {
						logp.Warn("unexpected fragment ID: %d != %d", f.id, id)
					}
					if f.offset != len(data) {
						logp.Warn("unexpected fragment offset for packet ID=%d: %d != %d", id, f.offset, len(data))
					}
					if len(data)+len(f.data) > maxReconstruct {
						logp.Warn("packet reconstruction would exceed limit ID=%d src=%s dst=%s", id, ipv4.SrcIP, ipv4.DstIP)
						return
					}
					data = append(data, f.data...)
				}
				d.fragments = d.fragments[:0]
				d.process(&packet, currentType)
				currentType = ipv4.Protocol.LayerType()
				current, ok = d.decoders[currentType]
				if !ok {
					logp.Debug("decoder", "no layer decoder for reconstructed fragments: %v (%[1]d)", currentType)
					break
				}
				continue
			}
		}

		done := d.process(&packet, currentType)
		if done {
			logp.Debug("decoder", "processed")
			break
		}

		// choose next decoding layer
		next, ok := d.decoders[nextType]
		if !ok {
			logp.Debug("decoder", "no next type: %v (%[1]d)", nextType)
			break
		}

		// jump to next layer
		current = next
		currentType = nextType
	}

	// add flow s.tats
	if d.flowID != nil {
		logp.Debug("decoder", "flow id flags: %v", d.flowID.Flags())
	}

	if d.flowID != nil && d.flowID.Flags() != 0 {
		flow := d.flows.Get(d.flowID)
		d.statPackets.Add(flow, 1)
		d.statBytes.Add(flow, uint64(ci.Length))
	}
}

func (d *Decoder) process(packet *protos.Packet, layerType gopacket.LayerType) (done bool) {
	withFlow := d.flowID != nil

	switch layerType {
	case layers.LayerTypeEthernet:
		if withFlow {
			d.flowID.AddEth(d.eth.SrcMAC, d.eth.DstMAC)
		}

	case layers.LayerTypeDot1Q:
		d1q := &d.d1q[d.stD1Q.i]
		d.stD1Q.next()
		if withFlow {
			d.flowID.AddVLan(d1q.VLANIdentifier)
		}

	case layers.LayerTypeIPv4:
		logp.Debug("decoder", "IPv4 packet")
		ip4 := &d.ip4[d.stIP4.i]
		d.stIP4.next()

		if withFlow {
			d.flowID.AddIPv4(ip4.SrcIP, ip4.DstIP)
		}

		packet.Tuple.SrcIP = ip4.SrcIP
		packet.Tuple.DstIP = ip4.DstIP
		packet.Tuple.IPLength = 4

	case layers.LayerTypeIPv6:
		logp.Debug("decoder", "IPv6 packet")
		ip6 := &d.ip6[d.stIP6.i]
		d.stIP6.next()

		if withFlow {
			d.flowID.AddIPv6(ip6.SrcIP, ip6.DstIP)
		}

		packet.Tuple.SrcIP = ip6.SrcIP
		packet.Tuple.DstIP = ip6.DstIP
		packet.Tuple.IPLength = 16

	case layers.LayerTypeICMPv4:
		logp.Debug("decoder", "ICMPv4 packet")
		d.onICMPv4(packet)
		return true

	case layers.LayerTypeICMPv6:
		logp.Debug("decoder", "ICMPv6 packet")
		d.onICMPv6(packet)
		return true

	case layers.LayerTypeUDP:
		logp.Debug("decoder", "UDP packet")
		d.onUDP(packet)
		return true

	case layers.LayerTypeTCP:
		logp.Debug("decoder", "TCP packet")
		d.onTCP(packet)
		return true
	}

	return false
}

func (d *Decoder) onICMPv4(packet *protos.Packet) {
	if d.flowID != nil {
		flow := d.flows.Get(d.flowID)
		d.icmpV4TypeCode.Set(flow, uint64(d.icmp4.TypeCode))
	}

	if d.icmp4Proc != nil {
		packet.Payload = d.icmp4.Payload
		packet.Tuple.ComputeHashables()
		d.icmp4Proc.ProcessICMPv4(d.flowID, &d.icmp4, packet)
	}
}

func (d *Decoder) onICMPv6(packet *protos.Packet) {
	if d.flowID != nil {
		flow := d.flows.Get(d.flowID)
		d.icmpV6TypeCode.Set(flow, uint64(d.icmp6.TypeCode))
	}

	if d.icmp6Proc != nil {
		// google/gopacket treats the first four bytes
		// after the typo, code and checksum as part of
		// the payload. So drop those bytes.
		// See https://github.com/google/gopacket/pull/423/
		d.icmp6.Payload = d.icmp6.Payload[4:]
		packet.Payload = d.icmp6.Payload
		packet.Tuple.ComputeHashables()
		d.icmp6Proc.ProcessICMPv6(d.flowID, &d.icmp6, packet)
	}
}

func (d *Decoder) onUDP(packet *protos.Packet) {
	src := uint16(d.udp.SrcPort)
	dst := uint16(d.udp.DstPort)

	id := d.flowID
	if id != nil {
		d.flowID.AddUDP(src, dst)
	}

	packet.Tuple.SrcPort = src
	packet.Tuple.DstPort = dst
	packet.Payload = d.udp.Payload
	packet.Tuple.ComputeHashables()

	d.udpProc.Process(id, packet)
}

func (d *Decoder) onTCP(packet *protos.Packet) {
	src := uint16(d.tcp.SrcPort)
	dst := uint16(d.tcp.DstPort)

	id := d.flowID
	if id != nil {
		id.AddTCP(src, dst)
	}

	packet.Tuple.SrcPort = src
	packet.Tuple.DstPort = dst
	packet.Payload = d.tcp.Payload

	if id == nil && len(packet.Payload) == 0 && !d.tcp.FIN {
		// We have no use for this atm.
		logp.Debug("decoder", "Ignore empty non-FIN packet")
		return
	}
	packet.Tuple.ComputeHashables()
	d.tcpProc.Process(id, &d.tcp, packet)
}
