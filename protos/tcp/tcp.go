package tcp

import (
	"fmt"
	"packetbeat/common"
	"packetbeat/config"
	"packetbeat/logp"
	"packetbeat/protos"
	"strings"
	"time"

	"github.com/packetbeat/gopacket"
	"github.com/packetbeat/gopacket/layers"
)

const TCP_STREAM_EXPIRY = 10 * 1e9
const TCP_STREAM_HASH_SIZE = 2 ^ 16
const TCP_MAX_DATA_IN_STREAM = 10 * 1e6

const (
	TcpDirectionReverse  = 0
	TcpDirectionOriginal = 1
)

var __id uint32 = 0

func GetId() uint32 {
	__id += 1
	return __id
}

// Config

var tcpStreamsMap = make(map[common.HashableIpPortTuple]*TcpStream, TCP_STREAM_HASH_SIZE)
var tcpPortMap map[uint16]protos.Protocol

func decideProtocol(tuple *common.IpPortTuple) protos.Protocol {
	protocol, exists := tcpPortMap[tuple.Src_port]
	if exists {
		return protocol
	}

	protocol, exists = tcpPortMap[tuple.Dst_port]
	if exists {
		return protocol
	}

	return protos.UnknownProtocol
}

type TcpStream struct {
	id       uint32
	tuple    *common.IpPortTuple
	timer    *time.Timer
	protocol protos.Protocol
	tcptuple common.TcpTuple

	lastSeq [2]uint32

	// protocols private data
	Data protos.ProtocolData
}

func (stream *TcpStream) AddPacket(pkt *protos.Packet, tcphdr *layers.TCP, original_dir uint8) {

	// create/reset timer
	if stream.timer != nil {
		stream.timer.Stop()
	}
	stream.timer = time.AfterFunc(TCP_STREAM_EXPIRY, func() { stream.Expire() })

	mod := protos.Protos.Get(stream.protocol)
	if mod == nil {
		logp.Debug("tcp", "Ignoring protocol for which we have no module loaded: %s", stream.protocol)
		return
	}

	if len(pkt.Payload) > 0 {
		stream.Data = mod.Parse(pkt, &stream.tcptuple, original_dir, stream.Data)
	}

	if tcphdr.FIN {
		stream.Data = mod.ReceivedFin(&stream.tcptuple, original_dir, stream.Data)
	}
}

func (stream *TcpStream) GapInStream(original_dir uint8) {
	mod := protos.Protos.Get(stream.protocol)
	stream.Data = mod.GapInStream(&stream.tcptuple, original_dir, stream.Data)
}

func (stream *TcpStream) Expire() {

	logp.Debug("mem", "Tcp stream expired")

	// de-register from dict
	delete(tcpStreamsMap, stream.tuple.Hashable())

	// nullify to help the GC
	stream.Data = nil
}

func TcpSeqBefore(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) < 0
}

func TcpSeqBeforeEq(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) <= 0
}

func FollowTcp(tcphdr *layers.TCP, pkt *protos.Packet) {
	stream, exists := tcpStreamsMap[pkt.Tuple.Hashable()]
	var original_dir uint8 = TcpDirectionOriginal
	created := false
	if !exists {
		stream, exists = tcpStreamsMap[pkt.Tuple.RevHashable()]
		if !exists {
			protocol := decideProtocol(&pkt.Tuple)
			if protocol == protos.UnknownProtocol {
				// don't follow
				return
			}
			logp.Debug("tcp", "Stream doesn't exists, creating new")

			// create
			stream = &TcpStream{id: GetId(), tuple: &pkt.Tuple, protocol: protocol}
			stream.tcptuple = common.TcpTupleFromIpPort(stream.tuple, stream.id)
			tcpStreamsMap[pkt.Tuple.Hashable()] = stream
			created = true
		} else {
			original_dir = TcpDirectionReverse
		}
	}
	tcp_start_seq := tcphdr.Seq
	tcp_seq := tcp_start_seq + uint32(len(pkt.Payload))

	logp.Debug("tcp", "pkt.start_seq=%v pkt.last_seq=%v stream.last_seq=%v (len=%d)",
		tcp_start_seq, tcp_seq, stream.lastSeq[original_dir], len(pkt.Payload))

	if len(pkt.Payload) > 0 &&
		stream.lastSeq[original_dir] != 0 {

		if TcpSeqBeforeEq(tcp_seq, stream.lastSeq[original_dir]) {

			logp.Debug("tcp", "Ignoring what looks like a retrasmitted segment. pkt.seq=%v len=%v stream.seq=%v",
				tcphdr.Seq, len(pkt.Payload), stream.lastSeq[original_dir])
			return
		}

		if TcpSeqBefore(stream.lastSeq[original_dir], tcp_start_seq) {
			logp.Debug("tcp", "Gap in tcp stream. last_seq: %d, seq: %d", stream.lastSeq[original_dir], tcp_start_seq)
			if !created {
				stream.GapInStream(original_dir)
				// drop stream
				stream.Expire()
				return
			}
		}
	}
	stream.lastSeq[original_dir] = tcp_seq

	stream.AddPacket(pkt, tcphdr, original_dir)
}

func PrintTcpMap() {
	fmt.Printf("Streams in memory:")
	for _, stream := range tcpStreamsMap {
		fmt.Printf(" %d", stream.id)
	}
	fmt.Printf("\n")

	fmt.Printf("Streams dict: %v", tcpStreamsMap)
}

func configToPortsMap(protocols map[string]config.Protocol) map[uint16]protos.Protocol {
	var res = map[uint16]protos.Protocol{}

	var proto protos.Protocol
	for proto = protos.UnknownProtocol + 1; int(proto) < len(protos.ProtocolNames); proto++ {

		protoConfig, exists := protocols[protos.ProtocolNames[proto]]
		if !exists {
			// skip
			continue
		}

		for _, port := range protoConfig.Ports {
			res[uint16(port)] = proto
		}
	}

	return res
}

func ConfigToFilter(protocols map[string]config.Protocol) string {

	res := []string{}

	for _, protoConfig := range protocols {
		for _, port := range protoConfig.Ports {
			res = append(res, fmt.Sprintf("port %d", port))
		}
	}

	return strings.Join(res, " or ")
}

func TcpInit(protocols map[string]config.Protocol) error {
	tcpPortMap = configToPortsMap(protocols)

	logp.Debug("tcp", "Port map: %v", tcpPortMap)

	return nil
}

type DecoderStruct struct {
	Parser *gopacket.DecodingLayerParser

	sll     layers.LinuxSLL
	lo      layers.Loopback
	eth     layers.Ethernet
	ip4     layers.IPv4
	ip6     layers.IPv6
	tcp     layers.TCP
	payload gopacket.Payload
	decoded []gopacket.LayerType
}

func CreateDecoder(datalink layers.LinkType) (*DecoderStruct, error) {
	var d DecoderStruct

	logp.Debug("pcapread", "Layer type: %s", datalink.String())

	switch datalink {

	case layers.LinkTypeLinuxSLL:
		d.Parser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeLinuxSLL,
			&d.sll, &d.ip4, &d.ip6, &d.tcp, &d.payload)

	case layers.LinkTypeEthernet:
		d.Parser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeEthernet,
			&d.eth, &d.ip4, &d.ip6, &d.tcp, &d.payload)

	case layers.LinkTypeNull: // loopback on OSx
		d.Parser = gopacket.NewDecodingLayerParser(
			layers.LayerTypeLoopback,
			&d.lo, &d.ip4, &d.ip6, &d.tcp, &d.payload)

	default:
		return nil, fmt.Errorf("Unsuported link type: %s", datalink.String())

	}

	d.decoded = []gopacket.LayerType{}

	return &d, nil
}

func (decoder *DecoderStruct) DecodePacketData(data []byte, ci *gopacket.CaptureInfo) {

	var err error
	var packet protos.Packet

	err = decoder.Parser.DecodeLayers(data, &decoder.decoded)
	if err != nil {
		logp.Debug("pcapread", "Decoding error: %s", err)
		return
	}

	has_tcp := false

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

		case gopacket.LayerTypePayload:
			packet.Payload = decoder.payload
		}
	}

	if !has_tcp {
		logp.Debug("pcapread", "No TCP header found in message")
		return
	}

	if len(packet.Payload) == 0 && !decoder.tcp.FIN {
		// We have no use for this atm.
		logp.Debug("pcapread", "Ignore empty non-FIN packet")
		return
	}

	packet.Ts = ci.Timestamp

	packet.Tuple.ComputeHashebles()
	FollowTcp(&decoder.tcp, &packet)
}
