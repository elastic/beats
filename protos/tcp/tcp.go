package tcp

import (
	"fmt"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"

	"github.com/elastic/packetbeat/protos"

	"github.com/tsg/gopacket/layers"
)

const TCP_STREAM_EXPIRY = 10 * 1e9
const TCP_STREAM_HASH_SIZE = 2 ^ 16
const TCP_MAX_DATA_IN_STREAM = 10 * 1e6

const (
	TcpDirectionReverse  = 0
	TcpDirectionOriginal = 1
)

type Tcp struct {
	id         uint32
	streamsMap map[common.HashableIpPortTuple]*TcpStream
	portMap    map[uint16]protos.Protocol
	protocols  protos.Protocols
}

type Processor interface {
	Process(tcphdr *layers.TCP, pkt *protos.Packet)
}

func (tcp *Tcp) getId() uint32 {
	tcp.id += 1
	return tcp.id
}

func (tcp *Tcp) decideProtocol(tuple *common.IpPortTuple) protos.Protocol {
	protocol, exists := tcp.portMap[tuple.Src_port]
	if exists {
		return protocol
	}

	protocol, exists = tcp.portMap[tuple.Dst_port]
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
	tcp      *Tcp

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

	mod := stream.tcp.protocols.GetTcp(stream.protocol)
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

func (stream *TcpStream) GapInStream(original_dir uint8, nbytes int) (drop bool) {
	mod := stream.tcp.protocols.GetTcp(stream.protocol)
	stream.Data, drop = mod.GapInStream(&stream.tcptuple, original_dir, nbytes, stream.Data)
	return drop
}

func (stream *TcpStream) Expire() {

	logp.Debug("mem", "Tcp stream expired")

	// de-register from dict
	delete(stream.tcp.streamsMap, stream.tuple.Hashable())

	// nullify to help the GC
	stream.Data = nil
}

func TcpSeqBefore(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) < 0
}

func TcpSeqBeforeEq(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) <= 0
}

func (tcp *Tcp) Process(tcphdr *layers.TCP, pkt *protos.Packet) {

	// This Recover should catch all exceptions in
	// protocol modules.
	defer logp.Recover("Process tcp exception")

	stream, exists := tcp.streamsMap[pkt.Tuple.Hashable()]
	var original_dir uint8 = TcpDirectionOriginal
	created := false
	if !exists {
		stream, exists = tcp.streamsMap[pkt.Tuple.RevHashable()]
		if !exists {
			protocol := tcp.decideProtocol(&pkt.Tuple)
			if protocol == protos.UnknownProtocol {
				// don't follow
				return
			}
			logp.Debug("tcp", "Stream doesn't exists, creating new")

			// create
			stream = &TcpStream{id: tcp.getId(), tuple: &pkt.Tuple, protocol: protocol, tcp: tcp}
			stream.tcptuple = common.TcpTupleFromIpPort(stream.tuple, stream.id)
			tcp.streamsMap[pkt.Tuple.Hashable()] = stream
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
			if !created {
				logp.Debug("tcp", "Gap in tcp stream. last_seq: %d, seq: %d", stream.lastSeq[original_dir], tcp_start_seq)
				drop := stream.GapInStream(original_dir,
					int(tcp_start_seq-stream.lastSeq[original_dir]))
				if drop {
					logp.Debug("tcp", "Dropping stream because of gap")
					stream.Expire()
				}
			}
		}
	}
	stream.lastSeq[original_dir] = tcp_seq

	stream.AddPacket(pkt, tcphdr, original_dir)
}

func (tcp *Tcp) PrintTcpMap() {
	fmt.Printf("Streams in memory:")
	for _, stream := range tcp.streamsMap {
		fmt.Printf(" %d", stream.id)
	}
	fmt.Printf("\n")

	fmt.Printf("Streams dict: %v", tcp.streamsMap)
}

func buildPortsMap(plugins map[protos.Protocol]protos.TcpProtocolPlugin) (map[uint16]protos.Protocol, error) {
	var res = map[uint16]protos.Protocol{}

	for proto, protoPlugin := range plugins {
		for _, port := range protoPlugin.GetPorts() {
			old_proto, exists := res[uint16(port)]
			if exists {
				if old_proto == proto {
					continue
				}
				return nil, fmt.Errorf("Duplicate port (%d) exists in %s and %s protocols",
					port, old_proto, proto)
			}
			res[uint16(port)] = proto
		}
	}

	return res, nil
}

// Creates and returns a new Tcp.
func NewTcp(p protos.Protocols) (*Tcp, error) {
	portMap, err := buildPortsMap(p.GetAllTcp())
	if err != nil {
		return nil, err
	}

	tcp := &Tcp{protocols: p, portMap: portMap}
	tcp.streamsMap = make(map[common.HashableIpPortTuple]*TcpStream, TCP_STREAM_HASH_SIZE)
	logp.Debug("tcp", "Port map: %v", portMap)

	return tcp, nil
}
