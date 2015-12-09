package tcp

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"

	"github.com/tsg/gopacket/layers"
)

const TCP_MAX_DATA_IN_STREAM = 10 * (1 << 20)

const (
	TcpDirectionReverse  = 0
	TcpDirectionOriginal = 1
)

type Tcp struct {
	id        uint32
	streams   *common.Cache
	portMap   map[uint16]protos.Protocol
	protocols protos.Protocols
}

type Processor interface {
	Process(tcphdr *layers.TCP, pkt *protos.Packet)
}

var (
	debugf  = logp.MakeDebug("tcp")
	isDebug = false
)

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

func (tcp *Tcp) getStream(k common.HashableIpPortTuple) *TcpStream {
	v := tcp.streams.Get(k)
	if v != nil {
		return v.(*TcpStream)
	}
	return nil
}

type TcpStream struct {
	id       uint32
	tuple    *common.IpPortTuple
	protocol protos.Protocol
	tcptuple common.TcpTuple
	tcp      *Tcp

	lastSeq [2]uint32

	// protocols private data
	data protos.ProtocolData
}

func (stream *TcpStream) String() string {
	return fmt.Sprintf("TcpStream id[%d] tuple[%s] protocol[%s] lastSeq[%d %d]",
		stream.id, stream.tuple, stream.protocol, stream.lastSeq[0], stream.lastSeq[1])
}

func (stream *TcpStream) addPacket(pkt *protos.Packet, tcphdr *layers.TCP, original_dir uint8) {
	mod := stream.tcp.protocols.GetTcp(stream.protocol)
	if mod == nil {
		if isDebug {
			debugf("Ignoring protocol for which we have no module loaded: %s",
				stream.protocol)
		}
		return
	}

	if len(pkt.Payload) > 0 {
		stream.data = mod.Parse(pkt, &stream.tcptuple, original_dir, stream.data)
	}

	if tcphdr.FIN {
		stream.data = mod.ReceivedFin(&stream.tcptuple, original_dir, stream.data)
	}
}

func (stream *TcpStream) gapInStream(original_dir uint8, nbytes int) (drop bool) {
	mod := stream.tcp.protocols.GetTcp(stream.protocol)
	stream.data, drop = mod.GapInStream(&stream.tcptuple, original_dir, nbytes, stream.data)
	return drop
}

func tcpSeqBefore(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) < 0
}

func tcpSeqBeforeEq(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) <= 0
}

func (tcp *Tcp) Process(tcphdr *layers.TCP, pkt *protos.Packet) {

	// This Recover should catch all exceptions in
	// protocol modules.
	defer logp.Recover("Process tcp exception")

	stream := tcp.getStream(pkt.Tuple.Hashable())
	var original_dir uint8 = TcpDirectionOriginal
	created := false
	if stream == nil {
		stream = tcp.getStream(pkt.Tuple.RevHashable())
		if stream == nil {
			protocol := tcp.decideProtocol(&pkt.Tuple)
			if protocol == protos.UnknownProtocol {
				// don't follow
				return
			}

			timeout := time.Duration(0)
			mod := tcp.protocols.GetTcp(protocol)
			if mod != nil {
				timeout = mod.ConnectionTimeout()
			}

			if isDebug {
				debugf("Stream doesn't exist, creating new")
			}

			// create
			stream = &TcpStream{id: tcp.getId(), tuple: &pkt.Tuple, protocol: protocol, tcp: tcp}
			stream.tcptuple = common.TcpTupleFromIpPort(stream.tuple, stream.id)
			tcp.streams.PutWithTimeout(pkt.Tuple.Hashable(), stream, timeout)
			created = true
		} else {
			original_dir = TcpDirectionReverse
		}
	}
	tcp_start_seq := tcphdr.Seq
	tcp_seq := tcp_start_seq + uint32(len(pkt.Payload))

	if isDebug {
		debugf("pkt.start_seq=%v pkt.last_seq=%v stream.last_seq=%v (len=%d)",
			tcp_start_seq, tcp_seq, stream.lastSeq[original_dir], len(pkt.Payload))
	}

	if len(pkt.Payload) > 0 &&
		stream.lastSeq[original_dir] != 0 {

		if tcpSeqBeforeEq(tcp_seq, stream.lastSeq[original_dir]) {
			if isDebug {
				debugf("Ignoring what looks like a retransmitted segment. pkt.seq=%v len=%v stream.seq=%v",
					tcphdr.Seq, len(pkt.Payload), stream.lastSeq[original_dir])
			}
			return
		}

		if tcpSeqBefore(stream.lastSeq[original_dir], tcp_start_seq) {
			if !created {
				logp.Warn("Gap in tcp stream. last_seq: %d, seq: %d", stream.lastSeq[original_dir], tcp_start_seq)
				drop := stream.gapInStream(original_dir,
					int(tcp_start_seq-stream.lastSeq[original_dir]))
				if drop {
					if isDebug {
						debugf("Dropping stream because of gap")
					}
					tcp.streams.Delete(stream.tuple.Hashable())
				}
			}
		}
	}
	stream.lastSeq[original_dir] = tcp_seq

	stream.addPacket(pkt, tcphdr, original_dir)
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
	isDebug = logp.IsDebug("tcp")

	portMap, err := buildPortsMap(p.GetAllTcp())
	if err != nil {
		return nil, err
	}

	tcp := &Tcp{
		protocols: p,
		portMap:   portMap,
		streams: common.NewCache(
			protos.DefaultTransactionExpiration,
			protos.DefaultTransactionHashSize),
	}
	tcp.streams.StartJanitor(protos.DefaultTransactionExpiration)
	if isDebug {
		debugf("tcp", "Port map: %v", portMap)
	}

	return tcp, nil
}
