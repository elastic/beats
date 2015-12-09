package decoder

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/protos/udp"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
)

type DecoderStruct struct {
	Parser *gopacket.DecodingLayerParser

	sll     layers.LinuxSLL
	d1q     layers.Dot1Q
	lo      layers.Loopback
	eth     layers.Ethernet
	ip4     layers.IPv4
	ip6     layers.IPv6
	tcp     layers.TCP
	udp     layers.UDP
	payload gopacket.Payload
	decoded []gopacket.LayerType

	tcpProc tcp.Processor
	udpProc udp.Processor
}

// Creates and returns a new DecoderStruct.
func NewDecoder(datalink layers.LinkType, tcp tcp.Processor, udp udp.Processor) (*DecoderStruct, error) {
	d := DecoderStruct{tcpProc: tcp, udpProc: udp}

	logp.Debug("pcapread", "Layer type: %s", datalink.String())

	switch datalink {

	case layers.LinkTypeLinuxSLL:
		d.Parser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeLinuxSLL,
			&d.sll, &d.d1q, &d.ip4, &d.ip6, &d.tcp, &d.udp, &d.payload)

	case layers.LinkTypeEthernet:
		d.Parser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeEthernet,
			&d.eth, &d.d1q, &d.ip4, &d.ip6, &d.tcp, &d.udp, &d.payload)

	case layers.LinkTypeNull: // loopback on OSx
		d.Parser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeLoopback,
			&d.lo, &d.d1q, &d.ip4, &d.ip6, &d.tcp, &d.udp, &d.payload)

	default:
		return nil, fmt.Errorf("Unsupported link type: %s", datalink.String())

	}

	d.decoded = []gopacket.LayerType{}

	return &d, nil
}

func (decoder *DecoderStruct) DecodePacketData(data []byte, ci *gopacket.CaptureInfo) {

	var err error
	var packet protos.Packet

	err = decoder.Parser.DecodeLayers(data, &decoder.decoded)
	if err != nil {
		// Ignore UnsupportedLayerType errors that can occur while parsing
		// UDP packets.
		lastLayer := decoder.decoded[len(decoder.decoded)-1]
		_, unsupported := err.(gopacket.UnsupportedLayerType)
		if !(unsupported && lastLayer == layers.LayerTypeUDP) {
			logp.Debug("pcapread", "Decoding error: %s", err)
			return
		}
	}

	has_tcp := false
	has_udp := false

	for _, layerType := range decoder.decoded {
		switch layerType {
		case layers.LayerTypeIPv4:
			logp.Debug("ip", "IPv4 packet")

			packet.Tuple.Src_ip = decoder.ip4.SrcIP
			packet.Tuple.Dst_ip = decoder.ip4.DstIP
			packet.Tuple.Ip_length = 4

		case layers.LayerTypeIPv6:
			logp.Debug("ip", "IPv6 packet")

			packet.Tuple.Src_ip = decoder.ip6.SrcIP
			packet.Tuple.Dst_ip = decoder.ip6.DstIP
			packet.Tuple.Ip_length = 16

		case layers.LayerTypeTCP:
			logp.Debug("ip", "TCP packet")

			packet.Tuple.Src_port = uint16(decoder.tcp.SrcPort)
			packet.Tuple.Dst_port = uint16(decoder.tcp.DstPort)

			has_tcp = true

		case layers.LayerTypeUDP:
			logp.Debug("ip", "UDP packet")

			packet.Tuple.Src_port = uint16(decoder.udp.SrcPort)
			packet.Tuple.Dst_port = uint16(decoder.udp.DstPort)
			packet.Payload = decoder.udp.Payload

			has_udp = true

		case gopacket.LayerTypePayload:
			packet.Payload = decoder.payload
		}
	}

	packet.Ts = ci.Timestamp
	packet.Tuple.ComputeHashebles()

	if has_udp {
		decoder.udpProc.Process(&packet)
	} else if has_tcp {
		if len(packet.Payload) == 0 && !decoder.tcp.FIN {
			// We have no use for this atm.
			logp.Debug("pcapread", "Ignore empty non-FIN packet")
			return
		}

		decoder.tcpProc.Process(&decoder.tcp, &packet)
	}
}
