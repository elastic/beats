package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/packetbeat/gopacket"
	"github.com/packetbeat/gopacket/layers"
)

const TCP_STREAM_EXPIRY = 10 * 1e9
const TCP_STREAM_HASH_SIZE = 2 ^ 16
const TCP_MAX_DATA_IN_STREAM = 10 * 1e6

type CmdlineTuple struct {
	Src, Dst []byte
}

const (
	TcpDirectionReverse  = 0
	TcpDirectionOriginal = 1
)

type Packet struct {
	ts      time.Time
	tuple   IpPortTuple
	payload []byte
}

type TcpStream struct {
	id       uint32
	tuple    *IpPortTuple
	timer    *time.Timer
	protocol protocolType

	lastSeq [2]uint32

	httpData   [2]*HttpStream
	mysqlData  [2]*MysqlStream
	redisData  [2]*RedisStream
	pgsqlData  [2]*PgsqlStream
	thriftData [2]*ThriftStream
}

type Endpoint struct {
	Ip      string
	Port    uint16
	Name    string
	Cmdline string
	Proc    string
}

var __id uint32 = 0

func GetId() uint32 {
	__id += 1
	return __id
}

// Config
type tomlProtocol struct {
	Ports         []int
	Send_request  bool
	Send_response bool
}

var tcpStreamsMap = make(map[HashableIpPortTuple]*TcpStream, TCP_STREAM_HASH_SIZE)
var tcpPortMap map[uint16]protocolType

func decideProtocol(tuple *IpPortTuple) protocolType {
	protocol, exists := tcpPortMap[tuple.Src_port]
	if exists {
		return protocol
	}

	protocol, exists = tcpPortMap[tuple.Dst_port]
	if exists {
		return protocol
	}

	return UnknownProtocol
}

func (stream *TcpStream) AddPacket(pkt *Packet, tcphdr *layers.TCP, original_dir uint8) {

	// create/reset timer
	if stream.timer != nil {
		stream.timer.Stop()
	}
	stream.timer = time.AfterFunc(TCP_STREAM_EXPIRY, func() { stream.Expire() })

	switch stream.protocol {
	case HttpProtocol:
		if len(pkt.payload) > 0 {
			HttpMod.Parse(pkt, stream, original_dir)
		}

		if tcphdr.FIN {
			HttpMod.ReceivedFin(stream, original_dir)
		}

	case MysqlProtocol:
		if len(pkt.payload) > 0 {
			ParseMysql(pkt, stream, original_dir)
		}

	case RedisProtocol:
		if len(pkt.payload) > 0 {
			ParseRedis(pkt, stream, original_dir)
		}

	case PgsqlProtocol:
		if len(pkt.payload) > 0 {
			ParsePgsql(pkt, stream, original_dir)
		}

	case ThriftProtocol:
		if len(pkt.payload) > 0 {
			ThriftMod.Parse(pkt, stream, original_dir)
		}

		if tcphdr.FIN {
			ThriftMod.ReceivedFin(stream, original_dir)
		}
	}
}

func (stream *TcpStream) GapInStream(original_dir uint8) {
	switch stream.protocol {
	case PgsqlProtocol:
		GapInPgsqlStream(stream, original_dir)
		break
	}
}

func (stream *TcpStream) Expire() {

	DEBUG("mem", "Tcp stream expired")

	// de-register from dict
	delete(tcpStreamsMap, stream.tuple.raw)

	// nullify to help the GC
	stream.httpData = [2]*HttpStream{nil, nil}
	stream.mysqlData = [2]*MysqlStream{nil, nil}
	stream.redisData = [2]*RedisStream{nil, nil}
	stream.pgsqlData = [2]*PgsqlStream{nil, nil}
}

func TcpSeqBefore(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) < 0
}

func TcpSeqBeforeEq(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) <= 0
}

func FollowTcp(tcphdr *layers.TCP, pkt *Packet) {
	stream, exists := tcpStreamsMap[pkt.tuple.raw]
	var original_dir uint8 = TcpDirectionOriginal
	created := false
	if !exists {
		stream, exists = tcpStreamsMap[pkt.tuple.revRaw]
		if !exists {
			protocol := decideProtocol(&pkt.tuple)
			if protocol == UnknownProtocol {
				// don't follow
				return
			}
			DEBUG("tcp", "Stream doesn't exists, creating new")

			// create
			stream = &TcpStream{id: GetId(), tuple: &pkt.tuple, protocol: protocol}
			tcpStreamsMap[pkt.tuple.raw] = stream
			created = true
		} else {
			original_dir = TcpDirectionReverse
		}
	}
	tcp_start_seq := tcphdr.Seq
	tcp_seq := tcp_start_seq + uint32(len(pkt.payload))

	DEBUG("tcp", "pkt.start_seq=%v pkt.last_seq=%v stream.last_seq=%v (len=%d)",
		tcp_start_seq, tcp_seq, stream.lastSeq[original_dir], len(pkt.payload))

	if len(pkt.payload) > 0 &&
		stream.lastSeq[original_dir] != 0 {

		if TcpSeqBeforeEq(tcp_seq, stream.lastSeq[original_dir]) {

			DEBUG("tcp", "Ignoring what looks like a retrasmitted segment. pkt.seq=%v len=%v stream.seq=%v",
				tcphdr.Seq, len(pkt.payload), stream.lastSeq[original_dir])
			return
		}

		if TcpSeqBefore(stream.lastSeq[original_dir], tcp_start_seq) {
			DEBUG("tcp", "Gap in tcp stream. last_seq: %d, seq: %d", stream.lastSeq[original_dir], tcp_start_seq)
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

func configToPortsMap(config *tomlConfig) map[uint16]protocolType {
	var res = map[uint16]protocolType{}

	var proto protocolType
	for proto = UnknownProtocol + 1; int(proto) < len(protocolNames); proto++ {

		protoConfig, exists := config.Protocols[protocolNames[proto]]
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

func configToFilter(config *tomlConfig) string {

	res := []string{}

	for _, protoConfig := range config.Protocols {
		for _, port := range protoConfig.Ports {
			res = append(res, fmt.Sprintf("port %d", port))
		}
	}

	return strings.Join(res, " or ")
}

func TcpInit() error {
	tcpPortMap = configToPortsMap(&_Config)

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

	DEBUG("pcapread", "Layer type: %s", datalink.String())

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
	var packet Packet

	err = decoder.Parser.DecodeLayers(data, &decoder.decoded)
	if err != nil {
		DEBUG("pcapread", "Decoding error: %s", err)
		return
	}

	has_tcp := false

	for _, layerType := range decoder.decoded {
		switch layerType {
		case layers.LayerTypeIPv4:
			DEBUG("ip", "IPv4 packet")

			packet.tuple.Src_ip = decoder.ip4.SrcIP
			packet.tuple.Dst_ip = decoder.ip4.DstIP
			packet.tuple.ip_length = 4

		case layers.LayerTypeIPv6:
			DEBUG("ip", "IPv6 packet")

			packet.tuple.Src_ip = decoder.ip6.SrcIP
			packet.tuple.Dst_ip = decoder.ip6.DstIP
			packet.tuple.ip_length = 16

		case layers.LayerTypeTCP:
			DEBUG("ip", "TCP packet")

			packet.tuple.Src_port = uint16(decoder.tcp.SrcPort)
			packet.tuple.Dst_port = uint16(decoder.tcp.DstPort)

			has_tcp = true

		case gopacket.LayerTypePayload:
			packet.payload = decoder.payload
		}
	}

	if !has_tcp {
		DEBUG("pcapread", "No TCP header found in message")
		return
	}

	if len(packet.payload) == 0 && !decoder.tcp.FIN {
		// We have no use for this atm.
		DEBUG("pcapread", "Ignore empty non-FIN packet")
		return
	}

	packet.ts = ci.Timestamp

	packet.tuple.ComputeHashebles()
	FollowTcp(&decoder.tcp, &packet)
}
