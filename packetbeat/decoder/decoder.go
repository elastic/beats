package decoder

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/flows"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/icmp"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/protos/udp"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
)

var debugf = logp.MakeDebug("decoder")

type DecoderStruct struct {
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

	stD1Q, stIP4, stIP6 multiLayer

	icmp4Proc icmp.ICMPv4Processor
	icmp6Proc icmp.ICMPv6Processor
	tcpProc   tcp.Processor
	udpProc   udp.Processor

	flows       *flows.Flows
	statPackets *flows.Uint
	statBytes   *flows.Uint

	// hold current flow ID
	flowID              *flows.FlowID // buffer flowID among many calls
	flowIDBufferBacking [flows.SizeFlowIDMax]byte
}

const (
	netPacketsTotalCounter = "net_packets_total"
	netBytesTotalCounter   = "net_bytes_total"
)

// Creates and returns a new DecoderStruct.
func NewDecoder(
	f *flows.Flows,
	datalink layers.LinkType,
	icmp4 icmp.ICMPv4Processor,
	icmp6 icmp.ICMPv6Processor,
	tcp tcp.Processor,
	udp udp.Processor,
) (*DecoderStruct, error) {
	d := DecoderStruct{
		flows:     f,
		decoders:  make(map[gopacket.LayerType]gopacket.DecodingLayer),
		icmp4Proc: icmp4, icmp6Proc: icmp6, tcpProc: tcp, udpProc: udp}
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
		d.flowID = &flows.FlowID{}
	}

	defaultLayerTypes := []gopacket.DecodingLayer{
		&d.sll,             // LinuxSLL
		&d.eth,             // Ethernet
		&d.lo,              // loopback on OS X
		&d.stD1Q,           // VLAN
		&d.stIP4, &d.stIP6, // IP
		&d.icmp4, &d.icmp6, // ICMP
		&d.tcp, &d.udp, // TCP/UDP
	}
	d.AddLayers(defaultLayerTypes)

	debugf("Layer type: %s", datalink.String())

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
		return nil, fmt.Errorf("Unsupported link type: %s", datalink.String())
	}

	return &d, nil
}

func (d *DecoderStruct) SetTruncated() {
	d.truncated = true
}

func (d *DecoderStruct) AddLayer(layer gopacket.DecodingLayer) {
	for _, typ := range layer.CanDecode().LayerTypes() {
		d.decoders[typ] = layer
	}
}

func (d *DecoderStruct) AddLayers(layers []gopacket.DecodingLayer) {
	for _, layer := range layers {
		d.AddLayer(layer)
	}
}

func (d *DecoderStruct) OnPacket(data []byte, ci *gopacket.CaptureInfo) {
	defer logp.Recover("packet decoding failed")

	d.truncated = false

	current := d.linkLayerDecoder
	currentType := d.linkLayerType

	packet := protos.Packet{Ts: ci.Timestamp}

	debugf("decode packet data")
	processed := false

	if d.flowID != nil {
		d.flowID.Reset(d.flowIDBufferBacking[:0])

		// supress flow stats snapshots while processing packet
		d.flows.Lock()
		defer d.flows.Unlock()
	}

	for len(data) > 0 {
		err := current.DecodeFromBytes(data, d)
		if err != nil {
			logp.Info("packet decode failed with: %v", err)
			break
		}

		nextType := current.NextLayerType()
		data = current.LayerPayload()

		processed, err = d.process(&packet, currentType)
		if err != nil {
			logp.Info("Error processing packet: %v", err)
			break
		}
		if processed {
			break
		}

		// choose next decoding layer
		next, ok := d.decoders[nextType]
		if !ok {
			break
		}

		// jump to next layer
		current = next
		currentType = nextType
	}

	// add flow s.tats
	if d.flowID != nil {
		debugf("flow id flags: %v", d.flowID.Flags())
	}

	if d.flowID != nil && d.flowID.Flags() != 0 {
		flow := d.flows.Get(d.flowID)
		d.statPackets.Add(flow, 1)
		d.statBytes.Add(flow, uint64(ci.Length))
	}
}

func (d *DecoderStruct) process(
	packet *protos.Packet,
	layerType gopacket.LayerType,
) (bool, error) {
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
		debugf("IPv4 packet")
		ip4 := &d.ip4[d.stIP4.i]
		d.stIP4.next()

		if withFlow {
			d.flowID.AddIPv4(ip4.SrcIP, ip4.DstIP)
		}

		packet.Tuple.Src_ip = ip4.SrcIP
		packet.Tuple.Dst_ip = ip4.DstIP
		packet.Tuple.Ip_length = 4

	case layers.LayerTypeIPv6:
		debugf("IPv6 packet")
		ip6 := &d.ip6[d.stIP6.i]
		d.stIP6.next()

		if withFlow {
			d.flowID.AddIPv6(ip6.SrcIP, ip6.DstIP)
		}

		packet.Tuple.Src_ip = ip6.SrcIP
		packet.Tuple.Dst_ip = ip6.DstIP
		packet.Tuple.Ip_length = 16

	case layers.LayerTypeICMPv4:
		debugf("ICMPv4 packet")
		d.onICMPv4(packet)
		return true, nil

	case layers.LayerTypeICMPv6:
		debugf("ICMPv6 packet")
		d.onICMPv6(packet)
		return true, nil

	case layers.LayerTypeUDP:
		debugf("UDP packet")
		d.onUDP(packet)
		return true, nil

	case layers.LayerTypeTCP:
		debugf("TCP packet")
		d.onTCP(packet)
		return true, nil
	}

	return false, nil
}

func (d *DecoderStruct) onICMPv4(packet *protos.Packet) {
	if d.icmp4Proc != nil {
		packet.Payload = d.icmp4.Payload
		packet.Tuple.ComputeHashebles()
		d.icmp4Proc.ProcessICMPv4(d.flowID, &d.icmp4, packet)
	}
}

func (d *DecoderStruct) onICMPv6(packet *protos.Packet) {
	if d.icmp6Proc != nil {
		packet.Payload = d.icmp6.Payload
		packet.Tuple.ComputeHashebles()
		d.icmp6Proc.ProcessICMPv6(d.flowID, &d.icmp6, packet)
	}
}

func (d *DecoderStruct) onUDP(packet *protos.Packet) {
	src := uint16(d.udp.SrcPort)
	dst := uint16(d.udp.DstPort)

	id := d.flowID
	if id != nil {
		d.flowID.AddUDP(src, dst)
	}

	packet.Tuple.Src_port = src
	packet.Tuple.Dst_port = dst
	packet.Payload = d.udp.Payload
	packet.Tuple.ComputeHashebles()

	d.udpProc.Process(id, packet)
}

func (d *DecoderStruct) onTCP(packet *protos.Packet) {
	src := uint16(d.tcp.SrcPort)
	dst := uint16(d.tcp.DstPort)

	id := d.flowID
	if id != nil {
		id.AddTCP(src, dst)
	}

	packet.Tuple.Src_port = src
	packet.Tuple.Dst_port = dst
	packet.Payload = d.tcp.Payload

	if id == nil && len(packet.Payload) == 0 && !d.tcp.FIN {
		// We have no use for this atm.
		debugf("Ignore empty non-FIN packet")
		return
	}
	packet.Tuple.ComputeHashebles()
	d.tcpProc.Process(id, &d.tcp, packet)
}
