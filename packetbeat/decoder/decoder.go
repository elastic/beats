package decoder

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/icmp"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/protos/udp"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
)

type DecoderStruct struct {
	decoders         map[gopacket.LayerType]gopacket.DecodingLayer
	linkLayerDecoder gopacket.DecodingLayer
	linkLayerType    gopacket.LayerType

	sll       layers.LinuxSLL
	d1q       layers.Dot1Q
	lo        layers.Loopback
	eth       layers.Ethernet
	ip4       layers.IPv4
	ip6       layers.IPv6
	icmp4     layers.ICMPv4
	icmp6     layers.ICMPv6
	tcp       layers.TCP
	udp       layers.UDP
	truncated bool

	icmp4Proc icmp.ICMPv4Processor
	icmp6Proc icmp.ICMPv6Processor
	tcpProc   tcp.Processor
	udpProc   udp.Processor
}

// Creates and returns a new DecoderStruct.
func NewDecoder(
	datalink layers.LinkType,
	icmp4 icmp.ICMPv4Processor,
	icmp6 icmp.ICMPv6Processor,
	tcp tcp.Processor,
	udp udp.Processor,
) (*DecoderStruct, error) {
	d := DecoderStruct{
		decoders:  make(map[gopacket.LayerType]gopacket.DecodingLayer),
		icmp4Proc: icmp4, icmp6Proc: icmp6, tcpProc: tcp, udpProc: udp}

	defaultLayerTypes := []gopacket.DecodingLayer{
		&d.sll,                             // LinuxSLL
		&d.eth,                             // Ethernet
		&d.lo,                              // loopback on OS X
		&d.d1q,                             // VLAN
		&d.ip4, &d.ip6, &d.icmp4, &d.icmp6, // IP
		&d.tcp, &d.udp, // TCP/UDP
	}
	d.AddLayers(defaultLayerTypes)

	logp.Debug("pcapread", "Layer type: %s", datalink.String())

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

func (d *DecoderStruct) DecodePacketData(data []byte, ci *gopacket.CaptureInfo) {
	defer logp.Recover("packet decoding failed")

	d.truncated = false

	current := d.linkLayerDecoder
	currentType := d.linkLayerType

	packet := protos.Packet{Ts: ci.Timestamp}

	logp.Info("decode packet data")

	for len(data) > 0 {
		err := current.DecodeFromBytes(data, d)
		if err != nil {
			logp.Info("packet decode failed with: %v", err)
			break
		}

		if err := d.process(&packet, currentType); err != nil {
			logp.Info("Error processing packet: %v", err)
			break
		}

		nextType := current.NextLayerType()
		data = current.LayerPayload()

		// choose next decoding layer
		next, ok := d.decoders[nextType]
		if !ok {
			break
		}

		// jump to next layer
		current = next
		currentType = nextType
	}
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

func (d *DecoderStruct) process(
	packet *protos.Packet,
	layerType gopacket.LayerType,
) error {
	switch layerType {
	case layers.LayerTypeIPv4:
		logp.Debug("ip", "IPv4 packet")
		packet.Tuple.Src_ip = d.ip4.SrcIP
		packet.Tuple.Dst_ip = d.ip4.DstIP
		packet.Tuple.Ip_length = 4
	case layers.LayerTypeIPv6:
		logp.Debug("ip", "IPv6 packet")
		packet.Tuple.Src_ip = d.ip6.SrcIP
		packet.Tuple.Dst_ip = d.ip6.DstIP
		packet.Tuple.Ip_length = 16
	case layers.LayerTypeICMPv4:
		logp.Debug("ip", "ICMPv4 packet")
		packet.Payload = d.icmp4.Payload

		packet.Tuple.ComputeHashebles()
		d.icmp4Proc.ProcessICMPv4(&d.icmp4, packet)
	case layers.LayerTypeICMPv6:
		logp.Debug("ip", "ICMPv6 packet")
		packet.Payload = d.icmp6.Payload

		packet.Tuple.ComputeHashebles()
		d.icmp6Proc.ProcessICMPv6(&d.icmp6, packet)
	case layers.LayerTypeUDP:
		logp.Debug("ip", "UDP packet")
		packet.Tuple.Src_port = uint16(d.udp.SrcPort)
		packet.Tuple.Dst_port = uint16(d.udp.DstPort)
		packet.Payload = d.udp.Payload
		packet.Tuple.ComputeHashebles()
		d.udpProc.Process(packet)
	case layers.LayerTypeTCP:
		logp.Debug("ip", "TCP packet")
		packet.Tuple.Src_port = uint16(d.tcp.SrcPort)
		packet.Tuple.Dst_port = uint16(d.tcp.DstPort)
		packet.Payload = d.tcp.Payload

		if len(packet.Payload) == 0 && !d.tcp.FIN {
			// We have no use for this atm.
			logp.Debug("pcapread", "Ignore empty non-FIN packet")
			break
		}
		packet.Tuple.ComputeHashebles()
		d.tcpProc.Process(&d.tcp, packet)
	}

	return nil
}
