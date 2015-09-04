package decoder

import (
	"fmt"

	"github.com/elastic/libbeat/logp"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/tcp"
	"github.com/elastic/packetbeat/protos/udp"

	"github.com/google/gopacket"
	"github.com/google/gopacket/ip4defrag"
	"github.com/google/gopacket/layers"
)

type DecoderStruct struct {
	NetParser  *gopacket.DecodingLayerParser
	TcpParser  *gopacket.DecodingLayerParser
	UdpParser  *gopacket.DecodingLayerParser
	ipv4Defrag *ip4defrag.IPv4Defragmenter

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
		d.NetParser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeLinuxSLL,
			&d.sll, &d.d1q, &d.ip4, &d.ip6)

	case layers.LinkTypeEthernet:
		d.NetParser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeEthernet,
			&d.eth, &d.d1q, &d.ip4, &d.ip6)

	case layers.LinkTypeNull: // loopback on OSx
		d.NetParser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeLoopback,
			&d.lo, &d.d1q, &d.ip4, &d.ip6)

	default:
		return nil, fmt.Errorf("Unsuported link type: %s", datalink.String())

	}

	d.TcpParser = gopacket.NewDecodingLayerParser(layers.LayerTypeTCP,
		&d.tcp, &d.payload)
	d.UdpParser = gopacket.NewDecodingLayerParser(layers.LayerTypeUDP,
		&d.udp, &d.payload)

	d.decoded = []gopacket.LayerType{}
	d.ipv4Defrag = ip4defrag.NewIPv4Defragmenter()

	return &d, nil
}

func (decoder *DecoderStruct) DecodePacketData(data []byte, ci *gopacket.CaptureInfo) {

	var err error
	var packet protos.Packet

	fmt.Println("decode packet data")

	// parse up to transport layer first (eth, vlan, ip4/6 only)
	err = decoder.NetParser.DecodeLayers(data, &decoder.decoded)
	if err != nil {
		// Ignore UnsupportedLayerType errors that can occur while parsing
		// UDP packets.
		lastLayer := decoder.decoded[len(decoder.decoded)-1]
		var nextLayer gopacket.LayerType
		is_ip4_fragment := false
		switch lastLayer {
		case layers.LayerTypeIPv4:
			nextLayer = decoder.ip4.NextLayerType()
			is_ip4_fragment = nextLayer == gopacket.LayerTypeFragment
		case layers.LayerTypeIPv6:
			nextLayer = decoder.ip6.NextLayerType()
		}

		isUnsupported := !is_ip4_fragment &&
			nextLayer != layers.LayerTypeTCP &&
			nextLayer != layers.LayerTypeUDP
		if isUnsupported {
			logp.Debug("pcapread", "Decoding error: %s", err)
			fmt.Printf("decoding error: %s", err)
			return
		}

		if is_ip4_fragment {
			// beware: hacks lie ahead
			logp.Debug("pcapread", "IPv4 fragmented packet")
			fmt.Println("ip fragment")

			// send ip4 layer as is to decoder.
			ip4 := decoder.ip4 // need to copy on heap, as defragmenter retains pointer
			ip4_out, err := decoder.ipv4Defrag.DefragIPv4(&ip4)
			if err != nil {
				logp.Debug("pcapread", "Failed to defragment ipv4 packet: %s", err)
				return
			}

			// If no payload was returned, the packet was fully consumed by the
			// defragmenter. We should copy the payload now, as we do not know for
			// how long the defragmenter will hold on the packet.
			if ip4_out == nil {
				logp.Debug("pcapread", "packet retained by ipv4 defragmenter")

				ip4 := &decoder.ip4
				data := make([]byte, len(ip4.Payload)+len(ip4.Contents))
				copy(data, ip4.Contents)
				copy(data[len(data):], ip4.Payload)
				ip4.Contents = data[:len(ip4.Contents)]
				ip4.Payload = data[len(ip4.Contents):]
				return
			}

			decoder.ip4 = *ip4_out
		}
	}

	logp.Debug("pcapread", "continue")

	// find transport layer to be parsed
	fmt.Println("find transport layer")
	var nextLayer gopacket.LayerType
	var payload []byte
	lastLayer := decoder.decoded[len(decoder.decoded)-1]
	switch lastLayer {
	case layers.LayerTypeIPv4:
		logp.Debug("ip", "IPv4 packet")
		packet.Tuple.Src_ip = decoder.ip4.SrcIP
		packet.Tuple.Dst_ip = decoder.ip4.DstIP
		packet.Tuple.Ip_length = 4
		nextLayer = decoder.ip4.NextLayerType()
		payload = decoder.ip4.LayerPayload()
	case layers.LayerTypeIPv6:
		logp.Debug("ip", "IPv6 packet")

		packet.Tuple.Src_ip = decoder.ip6.SrcIP
		packet.Tuple.Dst_ip = decoder.ip6.DstIP
		packet.Tuple.Ip_length = 16
		nextLayer = decoder.ip6.NextLayerType()
		payload = decoder.ip6.LayerPayload()
	}

	var transpParser *gopacket.DecodingLayerParser = nil
	has_tcp := false
	has_udp := false
	switch nextLayer {
	case layers.LayerTypeTCP:
		transpParser = decoder.TcpParser
		has_tcp = true
	case layers.LayerTypeUDP:
		has_udp = true
		transpParser = decoder.UdpParser
	}
	if transpParser == nil {
		logp.Debug("ip", "unsupported transport protocol: %s", nextLayer)
		return
	}

	// parse transport layers
	decoded := decoder.decoded[:len(decoder.decoded)]
	err = transpParser.DecodeLayers(payload, &decoded)
	if err != nil {
		logp.Debug("ip", "failed to parse transport layer: %s", err)
	}

	decoder.decoded = append(decoder.decoded, decoded...)
	fmt.Printf("decoded: %v\n", decoded)
	for _, layerType := range decoded {
		switch layerType {
		case layers.LayerTypeTCP:
			logp.Debug("ip", "TCP packet")

			packet.Tuple.Src_port = uint16(decoder.tcp.SrcPort)
			packet.Tuple.Dst_port = uint16(decoder.tcp.DstPort)

		case layers.LayerTypeUDP:
			logp.Debug("ip", "UDP packet")

			packet.Tuple.Src_port = uint16(decoder.udp.SrcPort)
			packet.Tuple.Dst_port = uint16(decoder.udp.DstPort)
			packet.Payload = decoder.udp.Payload

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
