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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"

	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/protos"
)

const TCPMaxDataInStream = 10 * (1 << 20)

const (
	TCPDirectionReverse  = 0
	TCPDirectionOriginal = 1
)

type Processor interface {
	Process(flow *flows.FlowID, hdr *layers.TCP, pkt *protos.Packet)
}

type TCP struct {
	id           uint32
	streams      *common.Cache
	portMap      map[uint16]protos.Protocol
	protocols    protos.Protocols
	expiredConns expirationQueue

	metrics *inputMetrics
}

// Creates and returns a new Tcp.
func NewTCP(p protos.Protocols, id, device string, idx int) (*TCP, error) {
	isDebug = logp.IsDebug("tcp")

	portMap, err := buildPortsMap(p.GetAllTCP())
	if err != nil {
		return nil, err
	}

	tcp := &TCP{
		protocols: p,
		portMap:   portMap,
		metrics:   newInputMetrics(fmt.Sprintf("%s_%d", id, idx), device, portMap),
	}
	tcp.streams = common.NewCacheWithRemovalListener(
		protos.DefaultTransactionExpiration,
		protos.DefaultTransactionHashSize,
		tcp.removalListener)

	tcp.streams.StartJanitor(protos.DefaultTransactionExpiration)
	if isDebug {
		logp.Debug("tcp", "Port map: %v", portMap)
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

func (tcp *TCP) Process(id *flows.FlowID, tcphdr *layers.TCP, pkt *protos.Packet) {
	tcp.expiredConns.notifyAll()

	tcp.metrics.logFlags(tcphdr)

	stream, created := tcp.getStream(pkt)
	if stream.conn == nil {
		return
	}

	conn := stream.conn
	if id != nil {
		id.AddConnectionID(uint64(conn.id))
	}

	if isDebug {
		logp.Debug("tcp", "tcp flow id: %p", id)
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
		logp.Debug("tcp", "pkt.start_seq=%v pkt.last_seq=%v stream.last_seq=%v (len=%d)",
			tcpStartSeq, tcpSeq, lastSeq, len(pkt.Payload))
	}

	if len(pkt.Payload) != 0 {
		tcp.metrics.log(pkt)
	}
	if len(pkt.Payload) > 0 && lastSeq != 0 {
		if tcpSeqBeforeEq(tcpSeq, lastSeq) {
			if isDebug {
				logp.Debug("tcp", "Ignoring retransmitted segment. pkt.seq=%v len=%v stream.seq=%v",
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
			logp.Debug("tcp", "Gap in tcp stream. last_seq: %d, seq: %d, gap: %d", lastSeq, tcpStartSeq, gap)
			drop := stream.gapInStream(gap)
			if drop {
				if isDebug {
					logp.Debug("tcp", "Dropping connection state because of gap")
				}
				tcp.metrics.logDrop()

				// drop application layer connection state and
				// update stream_id for app layer analysers using stream_id for lookups
				conn.id = tcp.getID()
				conn.data = nil
			}

		case seqGT:
			// lastSeq > tcpStartSeq => overlapping TCP segment detected. shrink packet
			delta := lastSeq - tcpStartSeq

			if isDebug {
				logp.Debug("tcp", "Overlapping tcp segment. last_seq %d, seq: %d, delta: %d",
					lastSeq, tcpStartSeq, delta)
			}

			pkt.Payload = pkt.Payload[delta:]
			tcphdr.Seq += delta
			tcp.metrics.logOverlap()
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
		logp.Debug("tcp", "Connection src[%s:%d] dst[%s:%d] doesn't exist, creating new",
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

type seqCompare int

const (
	seqLT seqCompare = -1
	seqEq seqCompare = 0
	seqGT seqCompare = 1
)

var isDebug = false

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

func (tcp *TCP) Close() {
	if tcp.metrics == nil {
		return
	}
	tcp.metrics.close()
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

func (conn *TCPConnection) String() string {
	return fmt.Sprintf("TcpStream id[%d] tuple[%s] protocol[%s] lastSeq[%d %d]",
		conn.id, conn.tuple, conn.protocol, conn.lastSeq[0], conn.lastSeq[1])
}

type TCPStream struct {
	conn *TCPConnection
	dir  uint8
}

func (stream *TCPStream) addPacket(pkt *protos.Packet, tcphdr *layers.TCP) {
	conn := stream.conn
	mod := conn.tcp.protocols.GetTCP(conn.protocol)
	if mod == nil {
		if isDebug {
			protocol := conn.protocol
			logp.Debug("tcp", "Ignoring protocol for which we have no module loaded: %s",
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
				return nil, fmt.Errorf("duplicate port (%d) exists in %s and %s protocols",
					port, oldProto, proto)
			}
			res[uint16(port)] = proto
		}
	}

	return res, nil
}

type expirationQueue struct {
	mutex sync.Mutex
	conns []expiredConnection
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

type expiredConnection struct {
	mod  protos.ExpirationAwareTCPPlugin
	conn *TCPConnection
}

func (ec *expiredConnection) notify() {
	ec.mod.Expired(&ec.conn.tcptuple, ec.conn.data)
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	lastPacket time.Time

	// TCP flag counts.
	fin, syn, rst, psh, ack, urg, ece, cwr, ns *monitoring.Uint
	// Total number of headers, including zero length packets.
	headers *monitoring.Uint

	device         *monitoring.String // name of the device being monitored
	packets        *monitoring.Uint   // number of packets processed
	bytes          *monitoring.Uint   // number of bytes processed
	overlapped     *monitoring.Uint   // number of packets shrunk due to overlap
	dropped        *monitoring.Int    // number of packets dropped because of gaps
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime metrics.Sample     // histogram of the elapsed time between packet receipt and publication
}

// newInputMetrics returns an input metric for the TCP processor. If id or
// device is empty a nil inputMetric is returned.
func newInputMetrics(id, device string, ports map[uint16]protos.Protocol) *inputMetrics {
	if id == "" || device == "" {
		// An empty id signals to not record metrics,
		// while an empty device means we are reading
		// from a pcap file and no metrics are needed.
		return nil
	}
	devID := fmt.Sprintf("%s-tcp%s::%s", id, portList(ports), device)
	reg, unreg := inputmon.NewInputRegistry("tcp", devID, nil)
	out := &inputMetrics{
		unregister:     unreg,
		device:         monitoring.NewString(reg, "device"),
		packets:        monitoring.NewUint(reg, "received_events_total"),
		bytes:          monitoring.NewUint(reg, "received_bytes_total"),
		overlapped:     monitoring.NewUint(reg, "tcp_overlaps"),
		fin:            monitoring.NewUint(reg, "fin_flags_total"),
		syn:            monitoring.NewUint(reg, "syn_flags_total"),
		rst:            monitoring.NewUint(reg, "rst_flags_total"),
		psh:            monitoring.NewUint(reg, "psh_flags_total"),
		ack:            monitoring.NewUint(reg, "ack_flags_total"),
		urg:            monitoring.NewUint(reg, "urg_flags_total"),
		ece:            monitoring.NewUint(reg, "ece_flags_total"),
		cwr:            monitoring.NewUint(reg, "cwr_flags_total"),
		ns:             monitoring.NewUint(reg, "ns_flags_total"),
		headers:        monitoring.NewUint(reg, "received_headers_total"),
		dropped:        monitoring.NewInt(reg, "tcp.dropped_because_of_gaps"), // Name and type retained for compatibility.
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.device.Set(device)

	return out
}

// portList returns a dash-separated list of port numbers sorted ascending. A leading
// dash is prepended to the list if it is not empty.
func portList(m map[uint16]protos.Protocol) string {
	if len(m) == 0 {
		return ""
	}
	ports := make([]int, 0, len(m))
	for p := range m {
		ports = append(ports, int(p))
	}
	sort.Ints(ports)
	s := make([]string, len(ports)+1)
	for i, p := range ports {
		s[i+1] = strconv.FormatInt(int64(p), 10)
	}
	return strings.Join(s, "-")
}

// logFlags logs flag metric for the given packet header.
func (m *inputMetrics) logFlags(hdr *layers.TCP) {
	if m == nil {
		return
	}
	m.headers.Add(1)
	if hdr.FIN {
		m.fin.Add(1)
	}
	if hdr.SYN {
		m.syn.Add(1)
	}
	if hdr.RST {
		m.rst.Add(1)
	}
	if hdr.PSH {
		m.psh.Add(1)
	}
	if hdr.ACK {
		m.ack.Add(1)
	}
	if hdr.URG {
		m.urg.Add(1)
	}
	if hdr.ECE {
		m.ece.Add(1)
	}
	if hdr.CWR {
		m.cwr.Add(1)
	}
	if hdr.NS {
		m.ns.Add(1)
	}
}

// log logs metric for the given packet.
func (m *inputMetrics) log(pkt *protos.Packet) {
	if m == nil {
		return
	}
	m.processingTime.Update(time.Since(pkt.Ts).Nanoseconds())
	m.packets.Add(1)
	m.bytes.Add(uint64(len(pkt.Payload)))
	if !m.lastPacket.IsZero() {
		m.arrivalPeriod.Update(pkt.Ts.Sub(m.lastPacket).Nanoseconds())
	}
	m.lastPacket = pkt.Ts
}

// logOverlap logs metric for a overlapped packet.
func (m *inputMetrics) logOverlap() {
	if m == nil {
		return
	}
	m.overlapped.Add(1)
}

// logDrop logs metric for a dropped packet.
func (m *inputMetrics) logDrop() {
	if m == nil {
		return
	}
	m.dropped.Add(1)
}

func (m *inputMetrics) close() {
	if m == nil {
		return
	}
	m.unregister()
}
