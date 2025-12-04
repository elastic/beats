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
	"time"

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

	fragments fragmentCache

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

	allowMismatchedEth bool
	logger             *logp.Logger
}

const (
	netPacketsTotalCounter = "packets"
	netBytesTotalCounter   = "bytes"
	icmpV4TypeCodeValue    = "icmpV4TypeCode"
	icmpV6TypeCodeValue    = "icmpV6TypeCode"
)

// New creates and initializes a new packet decoder.
func New(f *flows.Flows, datalink layers.LinkType, icmp4 icmp.ICMPv4Processor, icmp6 icmp.ICMPv6Processor, tcp tcp.Processor, udp udp.Processor, allowMismatchedEth bool) (*Decoder, error) {
	d := Decoder{
		flows:     f,
		decoders:  make(map[gopacket.LayerType]gopacket.DecodingLayer),
		icmp4Proc: icmp4, icmp6Proc: icmp6, tcpProc: tcp, udpProc: udp,
		fragments:          fragmentCache{collected: make(map[fragmentKey]fragments), lastPurge: time.Now()},
		allowMismatchedEth: allowMismatchedEth,
		logger:             logp.NewLogger("decoder"),
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

	d.logger.Debugf("Layer type: %s", datalink)

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
	d.truncated = false

	current := d.linkLayerDecoder
	currentType := d.linkLayerType

	packet := protos.Packet{Ts: ci.Timestamp}

	d.logger.Debug("decode packet data")

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
			d.logger.Infof("packet decode failed with: %v", err)
			break
		}

		nextType := current.NextLayerType()
		data = current.LayerPayload()
		if nextType == gopacket.LayerTypeFragment {
			ipv4, ok := ipv4Layer(current)
			if !ok {
				// This should never happen. Log the issue and attempt to continue.
				// The process logic below will handle this if it can.
				d.logger.Warn("no IPv4 layer for fragment")
			} else {
				now := time.Now()
				const offsetMask = 1<<13 - 1 // https://datatracker.ietf.org/doc/html/rfc791#section-3.1
				key := fragmentKey{
					proto: ipv4.Protocol,
					id:    ipv4.Id,
				}
				if src := ipv4.SrcIP.To4(); src != nil {
					copy(key.src[:], src)
				}
				if dst := ipv4.DstIP.To4(); dst != nil {
					copy(key.dst[:], dst)
				}
				f := fragment{
					id:     ipv4.Id,
					offset: int(ipv4.FragOffset&offsetMask) * 8,
					data:   append(data[:0:0], data...), // Ensure that we are not aliasing data.
					more:   ipv4.Flags&layers.IPv4MoreFragments != 0,
					expire: now.Add(fragmentHold),
				}
				var more bool
				data, more, err = d.fragments.add(now, key, f)
				if err != nil {
					d.logger.Debugf("%v src=%s dst=%s", err, ipv4.SrcIP, ipv4.DstIP)
					return
				}
				if more {
					return
				}
				d.process(&packet, currentType)
				currentType = ipv4.Protocol.LayerType()
				current, ok = d.decoders[currentType]
				if !ok {
					d.logger.Debugf("no layer decoder for reconstructed fragments: %v (%[1]d)", currentType)
					break
				}
				continue
			}
		}

		done := d.process(&packet, currentType)
		if done {
			d.logger.Debugf("processed")
			break
		}

		// choose next decoding layer
		next, ok := d.decoders[nextType]
		if !ok {
			d.logger.Debugf("no next type: %v (%[1]d)", nextType)
			break
		}

		// jump to next layer
		current = next
		currentType = nextType
	}

	// add flow s.tats
	if d.flowID != nil {
		d.logger.Debugf("flow id flags: %v", d.flowID.Flags())
	}

	if d.flowID != nil && d.flowID.Flags() != 0 {
		flow := d.flows.Get(d.flowID)
		d.statPackets.Add(flow, 1)
		////nolint:gosec // G115: safe conversion, ci.Length is the size of the original packet and is therefore always positive
		d.statBytes.Add(flow, uint64(ci.Length))
	}
}

// fragmentCache is a TTL aware cache of IPv4 fragments to reassemble.
type fragmentCache struct {
	// lastPurge is the last time we attempted to purge expired fragments
	lastPurge time.Time

	// collected is the collections of fragments keyed on their
	// IPv4 packet ID field.
	collected map[fragmentKey]fragments
}

type fragmentKey struct {
	src   [4]byte
	dst   [4]byte
	proto layers.IPProtocol
	id    uint16
}

const (
	ipMaxLength        = 65535
	fragmentHold       = time.Second
	fragmentMaxPerFlow = 64
	fragmentMaxSets    = 512
)

// add adds a new fragment to the cache. The value of now is used to expire fragments
// and collections of fragments. If the fragment completes a set of fragments for
// reassembly, the payload of the reassembled packet is returned in data.
// If more packets are required to complete the reassembly of the packets in the
// fragments ID set, more is returned true. Expiries and oversize reassemblies are
// signaled via the returned error.
// The cache is purged of expired collections before add returns.
func (c *fragmentCache) add(now time.Time, k fragmentKey, f fragment) (data []byte, more bool, err error) {
	c.maybePurge(now)

	collected, ok := c.collected[k]
	if !ok {
		collected.expire = f.expire
	}

	// If this is a new tuple and we are at our limit of tuples, bail
	if len(c.collected)+1 >= fragmentMaxSets {
		delete(c.collected, k)
		return nil, false, fmt.Errorf("too many active fragment sets")
	}
	// If this tuple already has all fragments, bail
	if len(collected.fragments)+1 >= fragmentMaxPerFlow {
		delete(c.collected, k)
		return nil, false, fmt.Errorf("fragment limit exceeded for flow ID=%d", f.id)
	}
	// If the datagram would exceed the max, bail
	if collected.bytes+len(f.data) >= ipMaxLength {
		delete(c.collected, k)
		return nil, false, fmt.Errorf("fragment bytes limit exceeded for flow ID=%d", f.id)
	}

	collected.fragments = append(collected.fragments, f)
	collected.bytes += len(f.data)

	// Check whether we have all the fragments we need to do a reassembly.
	// Do the least amount of work possible
	if !f.more {
		collected.haveFinal = true
	}
	more = !collected.haveFinal
	if collected.haveFinal {
		sort.Slice(collected.fragments, func(i, j int) bool {
			return collected.fragments[i].offset < collected.fragments[j].offset
		})
		more = collected.fragments[0].offset != 0
		if !more {
			n := len(collected.fragments[0].data)
			for _, f := range collected.fragments[1:] {
				if f.offset != n {
					more = true
					break
				}
				n += len(f.data)
			}
		}
	}
	if more {
		c.collected[k] = collected
		return nil, true, nil
	}

	// Drop the fragments and do the reassembly.
	delete(c.collected, k)
	data = collected.fragments[0].data
	for _, f := range collected.fragments[1:] {
		data = append(data, f.data...)
	}
	return data, false, nil
}

// purge performs a cache expiry purge, removing all collected fragments
// that expired before now.
func (c *fragmentCache) maybePurge(now time.Time) {
	delta := now.Sub(c.lastPurge)
	if delta < fragmentHold {
		return
	}
	c.lastPurge = now

	for id, coll := range c.collected {
		if now.After(coll.expire) {
			delete(c.collected, id)
			continue
		}
	}
}

// fragments holds a collection of fragmented packets sharing an IPv4 packet ID.
type fragments struct {
	expire    time.Time
	fragments []fragment
	haveFinal bool
	bytes     int
}

// fragment is an IPv4 packet fragment.
type fragment struct {
	id     uint16
	offset int
	data   []byte
	more   bool
	expire time.Time
}

func (d *Decoder) process(packet *protos.Packet, layerType gopacket.LayerType) (done bool) {
	withFlow := d.flowID != nil

	switch layerType {
	case layers.LayerTypeEthernet:
		if withFlow && !d.allowMismatchedEth {
			d.flowID.AddEth(d.eth.SrcMAC, d.eth.DstMAC)
		}

	case layers.LayerTypeDot1Q:
		d1q := &d.d1q[d.stD1Q.i]
		d.stD1Q.next()
		if withFlow {
			d.flowID.AddVLan(d1q.VLANIdentifier)
		}

	case layers.LayerTypeIPv4:
		d.logger.Debugf("IPv4 packet")
		ip4 := &d.ip4[d.stIP4.i]
		d.stIP4.next()

		if withFlow {
			d.flowID.AddIPv4(ip4.SrcIP, ip4.DstIP)
		}

		packet.Tuple.SrcIP = ip4.SrcIP
		packet.Tuple.DstIP = ip4.DstIP
		packet.Tuple.IPLength = 4

	case layers.LayerTypeIPv6:
		d.logger.Debugf("IPv6 packet")
		ip6 := &d.ip6[d.stIP6.i]
		d.stIP6.next()

		if withFlow {
			d.flowID.AddIPv6(ip6.SrcIP, ip6.DstIP)
		}

		packet.Tuple.SrcIP = ip6.SrcIP
		packet.Tuple.DstIP = ip6.DstIP
		packet.Tuple.IPLength = 16

	case layers.LayerTypeICMPv4:
		d.logger.Debugf("ICMPv4 packet")
		d.onICMPv4(packet)
		return true

	case layers.LayerTypeICMPv6:
		d.logger.Debugf("ICMPv6 packet")
		d.onICMPv6(packet)
		return true

	case layers.LayerTypeUDP:
		d.logger.Debugf("UDP packet")
		d.onUDP(packet)
		return true

	case layers.LayerTypeTCP:
		d.logger.Debugf("TCP packet")
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
		d.logger.Debugf("Ignore empty non-FIN packet")
		return
	}
	packet.Tuple.ComputeHashables()
	d.tcpProc.Process(id, &d.tcp, packet)
}
