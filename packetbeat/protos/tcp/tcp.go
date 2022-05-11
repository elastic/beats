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

package tcp

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

const TCPMaxDataInStream = 10 * (1 << 20)

const (
	TCPDirectionReverse  = 0
	TCPDirectionOriginal = 1
)

type TCP struct {
	id           uint32
	streams      *common.Cache
	portMap      map[uint16]protos.Protocol
	protocols    protos.Protocols
	expiredConns expirationQueue
}

type expiredConnection struct {
	mod  protos.ExpirationAwareTCPPlugin
	conn *TCPConnection
}

type expirationQueue struct {
	mutex sync.Mutex
	conns []expiredConnection
}

type Processor interface {
	Process(flow *flows.FlowID, hdr *layers.TCP, pkt *protos.Packet)
}

var droppedBecauseOfGaps = monitoring.NewInt(nil, "tcp.dropped_because_of_gaps")

type seqCompare int

const (
	seqLT seqCompare = -1
	seqEq seqCompare = 0
	seqGT seqCompare = 1
)

var (
	debugf  = logp.MakeDebug("tcp")
	isDebug = false
)

func (tcp *TCP) getID() uint32 {
	tcp.id++
	return tcp.id
}

func (tcp *TCP) decideProtocol(tuple *common.IPPortTuple) protos.Protocol {
	protocol, exists := tcp.portMap[tuple.SrcPort]
	if exists {
		return protocol
	}

	protocol, exists = tcp.portMap[tuple.DstPort]
	if exists {
		return protocol
	}

	return protos.UnknownProtocol
}

func (tcp *TCP) findStream(k common.HashableIPPortTuple) *TCPConnection {
	v := tcp.streams.Get(k)
	if v != nil {
		return v.(*TCPConnection)
	}
	return nil
}

type TCPConnection struct {
	id       uint32
	tuple    *common.IPPortTuple
	protocol protos.Protocol
	tcptuple common.TCPTuple
	tcp      *TCP

	lastSeq [2]uint32

	// protocols private data
	data protos.ProtocolData
}

type TCPStream struct {
	conn *TCPConnection
	dir  uint8
}

func (conn *TCPConnection) String() string {
	return fmt.Sprintf("TcpStream id[%d] tuple[%s] protocol[%s] lastSeq[%d %d]",
		conn.id, conn.tuple, conn.protocol, conn.lastSeq[0], conn.lastSeq[1])
}

func (stream *TCPStream) addPacket(pkt *protos.Packet, tcphdr *layers.TCP) {
	conn := stream.conn
	mod := conn.tcp.protocols.GetTCP(conn.protocol)
	if mod == nil {
		if isDebug {
			protocol := conn.protocol
			debugf("Ignoring protocol for which we have no module loaded: %s",
				protocol)
		}
		return
	}

	if len(pkt.Payload) > 0 {
		conn.data = mod.Parse(pkt, &conn.tcptuple, stream.dir, conn.data)
	}

	if tcphdr.FIN {
		conn.data = mod.ReceivedFin(&conn.tcptuple, stream.dir, conn.data)
	}
}

func (stream *TCPStream) gapInStream(nbytes int) (drop bool) {
	conn := stream.conn
	mod := conn.tcp.protocols.GetTCP(conn.protocol)
	conn.data, drop = mod.GapInStream(&conn.tcptuple, stream.dir, nbytes, conn.data)
	return drop
}

func (tcp *TCP) Process(id *flows.FlowID, tcphdr *layers.TCP, pkt *protos.Packet) {
	// This Recover should catch all exceptions in
	// protocol modules.
	defer logp.Recover("Process tcp exception")

	tcp.expiredConns.notifyAll()

	stream, created := tcp.getStream(pkt)
	if stream.conn == nil {
		return
	}

	conn := stream.conn
	if id != nil {
		id.AddConnectionID(uint64(conn.id))
	}

	if isDebug {
		debugf("tcp flow id: %p", id)
	}

	if len(pkt.Payload) == 0 && !tcphdr.FIN {
		// return early if packet is not interesting. Still need to find/create
		// stream first in order to update the TCP stream timer
		return
	}

	tcpStartSeq := tcphdr.Seq
	tcpSeq := tcpStartSeq + uint32(len(pkt.Payload))
	lastSeq := conn.lastSeq[stream.dir]
	if isDebug {
		debugf("pkt.start_seq=%v pkt.last_seq=%v stream.last_seq=%v (len=%d)",
			tcpStartSeq, tcpSeq, lastSeq, len(pkt.Payload))
	}

	if len(pkt.Payload) > 0 && lastSeq != 0 {
		if tcpSeqBeforeEq(tcpSeq, lastSeq) {
			if isDebug {
				debugf("Ignoring retransmitted segment. pkt.seq=%v len=%v stream.seq=%v",
					tcphdr.Seq, len(pkt.Payload), lastSeq)
			}
			return
		}

		switch tcpSeqCompare(lastSeq, tcpStartSeq) {
		case seqLT: // lastSeq < tcpStartSeq => Gap in tcp stream detected
			if created {
				break
			}

			gap := int(tcpStartSeq - lastSeq)
			debugf("Gap in tcp stream. last_seq: %d, seq: %d, gap: %d", lastSeq, tcpStartSeq, gap)
			drop := stream.gapInStream(gap)
			if drop {
				if isDebug {
					debugf("Dropping connection state because of gap")
				}
				droppedBecauseOfGaps.Add(1)

				// drop application layer connection state and
				// update stream_id for app layer analysers using stream_id for lookups
				conn.id = tcp.getID()
				conn.data = nil
			}

		case seqGT:
			// lastSeq > tcpStartSeq => overlapping TCP segment detected. shrink packet
			delta := lastSeq - tcpStartSeq

			if isDebug {
				debugf("Overlapping tcp segment. last_seq %d, seq: %d, delta: %d",
					lastSeq, tcpStartSeq, delta)
			}

			pkt.Payload = pkt.Payload[delta:]
			tcphdr.Seq += delta
		}
	}

	conn.lastSeq[stream.dir] = tcpSeq
	stream.addPacket(pkt, tcphdr)
}

func (tcp *TCP) getStream(pkt *protos.Packet) (stream TCPStream, created bool) {
	if conn := tcp.findStream(pkt.Tuple.Hashable()); conn != nil {
		return TCPStream{conn: conn, dir: TCPDirectionOriginal}, false
	}

	if conn := tcp.findStream(pkt.Tuple.RevHashable()); conn != nil {
		return TCPStream{conn: conn, dir: TCPDirectionReverse}, false
	}

	protocol := tcp.decideProtocol(&pkt.Tuple)
	if protocol == protos.UnknownProtocol {
		// don't follow
		return TCPStream{}, false
	}

	var timeout time.Duration
	mod := tcp.protocols.GetTCP(protocol)
	if mod != nil {
		timeout = mod.ConnectionTimeout()
	}

	if isDebug {
		t := pkt.Tuple
		debugf("Connection src[%s:%d] dst[%s:%d] doesn't exist, creating new",
			t.SrcIP.String(), t.SrcPort,
			t.DstIP.String(), t.DstPort)
	}

	conn := &TCPConnection{
		id:       tcp.getID(),
		tuple:    &pkt.Tuple,
		protocol: protocol,
		tcp:      tcp,
	}
	conn.tcptuple = common.TCPTupleFromIPPort(conn.tuple, conn.id)
	tcp.streams.PutWithTimeout(pkt.Tuple.Hashable(), conn, timeout)
	return TCPStream{conn: conn, dir: TCPDirectionOriginal}, true
}

func tcpSeqCompare(seq1, seq2 uint32) seqCompare {
	i := int32(seq1 - seq2)
	switch {
	case i == 0:
		return seqEq
	case i < 0:
		return seqLT
	default:
		return seqGT
	}
}

func tcpSeqBeforeEq(seq1 uint32, seq2 uint32) bool {
	return int32(seq1-seq2) <= 0
}

func buildPortsMap(plugins map[protos.Protocol]protos.TCPPlugin) (map[uint16]protos.Protocol, error) {
	res := map[uint16]protos.Protocol{}

	for proto, protoPlugin := range plugins {
		for _, port := range protoPlugin.GetPorts() {
			oldProto, exists := res[uint16(port)]
			if exists {
				if oldProto == proto {
					continue
				}
				return nil, fmt.Errorf("Duplicate port (%d) exists in %s and %s protocols",
					port, oldProto, proto)
			}
			res[uint16(port)] = proto
		}
	}

	return res, nil
}

// Creates and returns a new Tcp.
func NewTCP(p protos.Protocols) (*TCP, error) {
	isDebug = logp.IsDebug("tcp")

	portMap, err := buildPortsMap(p.GetAllTCP())
	if err != nil {
		return nil, err
	}

	tcp := &TCP{
		protocols: p,
		portMap:   portMap,
	}
	tcp.streams = common.NewCacheWithRemovalListener(
		protos.DefaultTransactionExpiration,
		protos.DefaultTransactionHashSize,
		tcp.removalListener)

	tcp.streams.StartJanitor(protos.DefaultTransactionExpiration)
	if isDebug {
		debugf("tcp", "Port map: %v", portMap)
	}

	return tcp, nil
}

func (tcp *TCP) removalListener(_ common.Key, value common.Value) {
	conn := value.(*TCPConnection)
	mod := conn.tcp.protocols.GetTCP(conn.protocol)
	if mod != nil {
		awareMod, ok := mod.(protos.ExpirationAwareTCPPlugin)
		if ok {
			tcp.expiredConns.add(awareMod, conn)
		}
	}
}

func (ec *expiredConnection) notify() {
	ec.mod.Expired(&ec.conn.tcptuple, ec.conn.data)
}

func (eq *expirationQueue) add(mod protos.ExpirationAwareTCPPlugin, conn *TCPConnection) {
	eq.mutex.Lock()
	eq.conns = append(eq.conns, expiredConnection{
		mod:  mod,
		conn: conn,
	})
	eq.mutex.Unlock()
}

func (eq *expirationQueue) getExpired() (conns []expiredConnection) {
	eq.mutex.Lock()
	conns, eq.conns = eq.conns, nil
	eq.mutex.Unlock()
	return conns
}

func (eq *expirationQueue) notifyAll() {
	for _, expiration := range eq.getExpired() {
		expiration.notify()
	}
}
