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

package memcache

// Memcache TCP Protocol Plugin implementation

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
)

type tcpMemcache struct {
	tcpConfig
}

type tcpConfig struct {
	tcpTransTimeout time.Duration
}

type tcpConnectionData struct {
	streams    [2]*stream
	connection *connection
}

// memcache application layer streams
type stream struct {
	applayer.Stream
	parser parser
}

type connection struct {
	timer     *time.Timer
	requests  messageList
	responses messageList
}

type messageList struct {
	head *message
	tail *message
}

func ensureMemcacheConnection(private protos.ProtocolData) *tcpConnectionData {
	if private == nil {
		return &tcpConnectionData{}
	}

	priv, ok := private.(*tcpConnectionData)
	if !ok {
		logp.Warn("memcache connection data type error, create new one")
		return &tcpConnectionData{}
	}
	if priv == nil {
		logp.Warn("Unexpected: memcache TCP connection data not set, create new one")
		return &tcpConnectionData{}
	}
	return priv
}

func isMemcacheConnection(private protos.ProtocolData) bool {
	if private == nil {
		return false
	}

	_, ok := private.(*tcpConnectionData)
	return ok
}

func (mc *memcache) ConnectionTimeout() time.Duration {
	return mc.tcpTransTimeout
}

// Parse is called from TCP layer when payload data is available for parsing.
func (mc *memcache) Parse(
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseMemcache(TCP) exception")

	tcpConn := ensureMemcacheConnection(private)
	debug("memcache connection %p", tcpConn)
	tcpConn = mc.memcacheParseTCP(tcpConn, pkt, tcptuple, dir)
	if tcpConn == nil {
		// explicitly return nil if tcpConn equals nil so ProtocolData really is nil
		return nil
	}
	return tcpConn
}

func (mc *memcache) newStream(tcptuple *common.TCPTuple) *stream {
	s := &stream{}
	s.parser.init(&mc.config)
	s.Stream.Init(tcp.TCPMaxDataInStream)
	return s
}

func (mc *memcache) memcacheParseTCP(
	tcpConn *tcpConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TCPTuple,
	dir uint8,
) *tcpConnectionData {
	// assume we are in sync
	stream := tcpConn.streams[dir]
	if stream == nil {
		stream = mc.newStream(tcptuple)
		tcpConn.streams[dir] = stream
	}

	debug("add payload to stream(%p): %v", stream, dir)
	if err := stream.Append(pkt.Payload); err != nil {
		debug("%v, dropping TCP streams", err)
		mc.pushAllTCPTrans(tcpConn.connection)
		tcpConn.drop(dir)
		return nil
	}

	if tcpConn.connection == nil {
		tcpConn.connection = &connection{}
	} else {
		stopped := tcpConn.connection.timer.Stop()
		if !stopped {
			// timer was stopped by someone else, create new connection
			tcpConn.connection = &connection{}
		}
	}
	conn := tcpConn.connection

	for stream.Buf.Total() > 0 {
		debug("stream(%p) try to content", stream)
		msg, err := stream.parse(pkt.Ts)
		if err != nil {
			// parsing error, drop tcp stream and retry with next segement
			debug("Ignore Memcache message, drop tcp stream: %v", err)
			mc.pushAllTCPTrans(conn)
			tcpConn.drop(dir)
			return nil
		}

		if msg == nil {
			// wait for more data
			break
		}
		stream.reset()

		tuple := tcptuple.IPPort()
		err = mc.onTCPMessage(conn, tuple, dir, msg)
		if err != nil {
			logp.Warn("error processing memcache message: %s", err)
		}
	}

	conn.timer = time.AfterFunc(mc.tcpTransTimeout, func() {
		debug("connection=%p timed out", conn)
		mc.pushAllTCPTrans(conn)
	})

	return tcpConn
}

func (mc *memcache) onTCPMessage(
	conn *connection,
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	msg.Tuple = *tuple
	msg.Transport = applayer.TransportTCP
	msg.CmdlineTuple = mc.watcher.FindProcessesTupleTCP(tuple)

	if msg.IsRequest {
		return mc.onTCPRequest(conn, tuple, dir, msg)
	}
	return mc.onTCPResponse(conn, tuple, dir, msg)
}

func (mc *memcache) onTCPRequest(
	conn *connection,
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	requestSeenFirst := dir == tcp.TCPDirectionOriginal
	if requestSeenFirst {
		msg.Direction = applayer.NetOriginalDirection
	} else {
		msg.Direction = applayer.NetReverseDirection
	}

	debug("received memcached(tcp) request message=%p, tuple=%s",
		msg, msg.IsRequest, msg.Tuple)

	msg.isComplete = true
	waitResponse := msg.noreply ||
		(!msg.isBinary && msg.command.code != memcacheCmdQuit) ||
		(msg.isBinary && msg.opcode != opcodeQuitQ)
	if waitResponse {
		conn.requests.append(msg)
	} else {
		mc.onTCPTrans(msg, nil)
	}
	return nil
}

func (mc *memcache) onTCPResponse(
	conn *connection,
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	requestSeenFirst := dir == tcp.TCPDirectionReverse
	if requestSeenFirst {
		msg.Direction = applayer.NetOriginalDirection
	} else {
		msg.Direction = applayer.NetReverseDirection
	}

	debug("received memcached(tcp) response message=%p, tuple=%s",
		msg, msg.IsRequest, msg.Tuple)

	// try to merge response with last received response
	// (values and stats responses can be merged)
	prev := conn.responses.last()
	merged, err := tryMergeResponses(mc, prev, msg)
	if err != nil {
		return err
	}
	if merged {
		debug("response message got merged")
		msg = prev
	} else {
		conn.responses.append(msg)
	}
	if !msg.isComplete {
		return nil
	}

	debug("response message complete")

	return mc.correlateTCP(conn)
}

func (mc *memcache) correlateTCP(conn *connection) error {
	// merge requests with responses into transactions
	for !conn.responses.empty() {
		var requ *message
		resp := conn.responses.pop()

		for !conn.requests.empty() {
			requ = conn.requests.pop()
			if requ.isBinary != resp.isBinary {
				err := errMixOfBinaryAndText
				logp.Warn("%v", err)
				return err
			}

			// If requ and response belong to the same transaction, continue
			// merging them into one transaction
			sameTransaction := !requ.isBinary ||
				(requ.opaque == resp.opaque &&
					requ.opcode == resp.opcode)
			if sameTransaction {
				break
			}

			// check if we are missing a response or quiet message.
			// Quiet message only MAY get a response -> so we need
			// to clear message list from all quiet messages not having
			// received a response
			if requ.isBinary && !requ.isQuiet {
				note := noteNonQuietResponseOnly
				logp.Warn("%s", note)
				requ.AddNotes(note)
				unmatchedRequests.Add(1)
			}

			// send request
			debug("send single request=%p", requ)
			err := mc.onTCPTrans(requ, nil)
			if err != nil {
				logp.Warn("error processing memcache transaction: %s", err)
			}
			requ = nil
		}

		// Check if response without request. This should only happen when a TCP
		// stream is found (or after message gap) when we receive a response
		// without having seen a request.
		if requ == nil {
			debug("found orphan memcached response=%p", resp)
			resp.AddNotes(noteTransactionNoRequ)
			unmatchedResponses.Add(1)
		}

		debug("merge request=%p and response=%p", requ, resp)
		err := mc.onTCPTrans(requ, resp)
		if err != nil {
			logp.Warn("error processing memcache transaction: %s", err)
		}
		// continue processing more transactions (reporting error only)
	}

	return nil
}

func (mc *memcache) onTCPTrans(requ, resp *message) error {
	debug("received memcache(tcp) transaction")
	trans := newTransaction(requ, resp)
	return mc.finishTransaction(trans)
}

// GapInStream is called by TCP layer when a packets are missing from the tcp
// stream.
func (mc *memcache) GapInStream(
	tcptuple *common.TCPTuple,
	dir uint8, nbytes int,
	private protos.ProtocolData,
) (priv protos.ProtocolData, drop bool) {
	debug("memcache(tcp) stream gap detected")

	defer logp.Recover("GapInStream(memcache) exception")
	if !isMemcacheConnection(private) {
		return private, false
	}

	conn := private.(*tcpConnectionData)
	stream := conn.streams[dir]
	if stream == nil {
		debug("Inactive stream. Dropping connection state.")
		return private, true
	}

	parser := stream.parser
	msg := parser.message

	if msg != nil {
		if msg.IsRequest {
			msg.AddNotes(noteRequestPacketLoss)
		} else {
			msg.AddNotes(noteResponsePacketLoss)
		}
	}

	// If we are about to read binary data (length) encoded, but missing gab
	// does fully cover data area, we might be able to continue processing the
	// stream + transactions
	inData := parser.state == parseStateDataBinary ||
		parser.state == parseStateIncompleteDataBinary ||
		parser.state == parseStateData ||
		parser.state == parseStateIncompleteData
	if inData {
		if msg == nil {
			logp.WTF("parser message is nil on data load")
			return private, true
		}

		alreadyRead := stream.Buf.Len() - int(msg.bytesLost)
		dataRequired := int(msg.bytes) - alreadyRead
		if nbytes <= dataRequired {
			// yay, all bytes included in message binary data part.
			// just drop binary data part and recover parsing.
			if msg.isBinary {
				parser.state = parseStateIncompleteDataBinary
			} else {
				parser.state = parseStateIncompleteData
			}
			msg.bytesLost += uint(nbytes)
			return private, false
		}
	}

	// need to drop TCP stream. But try to publish all cached transactions first
	mc.pushAllTCPTrans(conn.connection)
	return private, true
}

// ReceivedFin is called by tcp layer when the FIN flag is seen in the TCP stream.
func (mc *memcache) ReceivedFin(
	tcptuple *common.TCPTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	// We rely on transaction timeout to publish all unfinished transactions.
	return private
}

func (mc *memcache) pushAllTCPTrans(conn *connection) {
	if conn == nil {
		return
	}

	// first let's try to send finished transactions
	// (unlikely we have some, though)
	mc.correlateTCP(conn)

	// only requests in map:
	debug("publish incomplete transactions")
	for !conn.requests.empty() {
		msg := conn.requests.pop()
		if !msg.isQuiet && !msg.noreply {
			msg.AddNotes(noteTransUnfinished)
			unfinishedTransactions.Add(1)
		}
		debug("push incomplete request=%p", msg)
		err := mc.onTCPTrans(msg, nil)
		if err != nil {
			logp.Warn("failed to publish unfinished transaction with %v", err)
		}
		// continue processing more transactions (reporting error only)
	}
}

func (private *tcpConnectionData) drop(dir uint8) {
	private.streams[dir] = nil
	private.streams[1-dir] = nil
}

func (stream *stream) reset() {
	parser := &stream.parser
	debug("consumed %v bytes", stream.Buf.BufferConsumed())
	stream.Stream.Reset()
	parser.reset()
}

func (stream *stream) parse(ts time.Time) (*message, error) {
	return stream.parser.parse(&stream.Buf, ts)
}

func (ml *messageList) append(msg *message) {
	if ml.tail == nil {
		ml.head = msg
	} else {
		ml.tail.next = msg
	}
	msg.next = nil
	ml.tail = msg
}

func (ml *messageList) empty() bool {
	return ml.head == nil
}

func (ml *messageList) pop() *message {
	if ml.head == nil {
		return nil
	}

	msg := ml.head
	ml.head = ml.head.next
	if ml.head == nil {
		ml.tail = nil
	}
	debug("new head=%p", ml.head)
	return msg
}

func (ml *messageList) last() *message {
	return ml.tail
}
