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

// Memcache UDP Protocol Plugin implementation.

import (
	"sync"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/streambuf"
	"github.com/elastic/beats/v8/libbeat/logp"

	"github.com/elastic/beats/v8/packetbeat/protos"
	"github.com/elastic/beats/v8/packetbeat/protos/applayer"
)

type udpMemcache struct {
	udpConfig      udpConfig
	udpConnections map[common.HashableIPPortTuple]*udpConnection
	udpExpTrans    udpExpTransList
}

type udpConfig struct {
	transTimeout time.Duration
}

type mcUDPHeader struct {
	requestID    uint16
	seqNumber    uint16
	numDatagrams uint16
}

type udpConnection struct {
	tuple        common.IPPortTuple
	transactions map[uint16]*udpTransaction
	memcache     *memcache
}

type udpTransaction struct {
	requestID uint16
	timer     *time.Timer
	next      *udpTransaction

	connection *udpConnection
	messages   [2]*udpMessage
	request    *message
	response   *message
}

// udpExpTransList holds a list of expired transactions for cleanup (manual gc)
// The list is filled by timeout handlers and cleaned by processing thread (ParseUdp)
// to deal with possible thread safety issues
//
// Note: only for cleanup. Transaction was published already,
//       as publishing is thread safe
type udpExpTransList struct {
	sync.Mutex
	head *udpTransaction
}

type udpMessage struct {
	numDatagrams uint16
	count        uint16
	isComplete   bool
	datagrams    [][]byte
}

func (mc *memcache) ParseUDP(pkt *protos.Packet) {
	defer logp.Recover("ParseMemcache(UDP) exception")

	buffer := streambuf.NewFixed(pkt.Payload)
	header, err := parseUDPHeader(buffer)
	if err != nil {
		debug("parsing memcache udp header failed")
		return
	}

	debug("new udp datagram requestId=%v, seqNumber=%v, numDatagrams=%v",
		header.requestID, header.seqNumber, header.numDatagrams)

	// find connection object based on ips and ports (forward->reverse connection)
	connection, dir := mc.getUDPConnection(&pkt.Tuple)
	debug("udp connection: %p", connection)

	// get udp transaction combining forward/reverse direction 'streams'
	// for current requestId
	trans := connection.udpTransactionForID(header.requestID)
	debug("udp transaction (id=%v): %p", header.requestID, trans)

	// Clean old transaction. We do the cleaning after potentially adding a new
	// transaction to the connection object, so connection object will not be
	// cleaned accidentally (not bad, but let's rather reuse it)
	expTrans := mc.udpExpTrans.steal()
	for expTrans != nil {
		tmp := expTrans.next
		expTrans.connection.killTransaction(expTrans)
		expTrans = tmp
	}

	// get UDP transaction stream combining datagram packets in transaction
	udpMsg := trans.udpMessageForDir(&header, dir)
	if udpMsg.numDatagrams != header.numDatagrams {
		logp.Warn("number of datagram mismatches in stream")
		connection.killTransaction(trans)
		return
	}

	// try to combine datagrams into complete memcached message
	payload := udpMsg.addDatagram(&header, buffer.Bytes())
	done := false
	if payload != nil {
		// parse memcached message
		msg, err := parseUDP(&mc.config, pkt.Ts, payload)
		if err != nil {
			logp.Warn("failed to parse memcached(UDP) message: %s", err)
			connection.killTransaction(trans)
			return
		}

		// apply memcached to transaction
		done, err = mc.onUDPMessage(trans, &pkt.Tuple, dir, msg)
		if err != nil {
			logp.Warn("error processing memcache message: %s", err)
			connection.killTransaction(trans)
			done = true
		}
	}
	if !done {
		trans.timer = time.AfterFunc(mc.udpConfig.transTimeout, func() {
			debug("transaction timeout -> forward")
			mc.onUDPTrans(trans)
			mc.udpExpTrans.push(trans)
		})
	}
}

func (mc *memcache) getUDPConnection(
	tuple *common.IPPortTuple,
) (*udpConnection, applayer.NetDirection) {
	connection := mc.udpConnections[tuple.Hashable()]
	if connection != nil {
		return connection, applayer.NetOriginalDirection
	}
	connection = mc.udpConnections[tuple.RevHashable()]
	if connection != nil {
		return connection, applayer.NetReverseDirection
	}

	connection = newUDPConnection(mc, tuple)
	mc.udpConnections[tuple.Hashable()] = connection
	return connection, applayer.NetOriginalDirection
}

func (mc *memcache) onUDPMessage(
	trans *udpTransaction,
	tuple *common.IPPortTuple,
	dir applayer.NetDirection,
	msg *message,
) (bool, error) {
	debug("received memcached(udp) message")

	if msg.IsRequest {
		msg.Direction = applayer.NetOriginalDirection
	} else {
		msg.Direction = applayer.NetReverseDirection
	}
	msg.Tuple = *tuple
	msg.Transport = applayer.TransportUDP
	msg.CmdlineTuple = mc.watcher.FindProcessesTupleUDP(tuple)

	done := false
	var err error
	if msg.IsRequest {
		msg.isComplete = true
		trans.request = msg
		waitResponse := msg.noreply ||
			(!msg.isBinary && msg.command.code != memcacheCmdQuit) ||
			(msg.isBinary && msg.opcode != opcodeQuitQ)
		done = !waitResponse
	} else {
		msg.isComplete = true
		trans.response = msg
	}

	done = done || (trans.request != nil && trans.response != nil)
	if done {
		err = mc.onUDPTrans(trans)
		trans.connection.killTransaction(trans)
	}
	return done, err
}

func (mc *memcache) onUDPTrans(udp *udpTransaction) error {
	debug("received memcache(udp) transaction")
	trans := newTransaction(udp.request, udp.response)
	return mc.finishTransaction(trans)
}

func newUDPConnection(mc *memcache, tuple *common.IPPortTuple) *udpConnection {
	c := &udpConnection{
		tuple:        *tuple,
		memcache:     mc,
		transactions: make(map[uint16]*udpTransaction),
	}
	return c
}

func (c *udpConnection) udpTransactionForID(requestID uint16) *udpTransaction {
	trans := c.transactions[requestID]
	if trans != nil && trans.timer != nil {
		stopped := trans.timer.Stop()
		if !stopped {
			logp.Warn("timer stopped while processing transaction -> create new transaction")
			trans = nil
		}
	}
	if trans == nil {
		trans = &udpTransaction{
			requestID:  requestID,
			connection: c,
		}
		c.transactions[requestID] = trans
	} else {
		trans.timer = nil
	}

	return trans
}

func (c *udpConnection) killTransaction(t *udpTransaction) {
	if t.timer != nil {
		t.timer.Stop()
	}

	if c.transactions[t.requestID] != t {
		// transaction was already replaced
		return
	}

	delete(c.transactions, t.requestID)
	if len(c.transactions) == 0 {
		delete(c.memcache.udpConnections, c.tuple.Hashable())
	}
}

func (lst *udpExpTransList) push(t *udpTransaction) {
	lst.Lock()
	defer lst.Unlock()
	t.next = lst.head
	lst.head = t
}

func (lst *udpExpTransList) steal() *udpTransaction {
	lst.Lock()
	t := lst.head
	lst.head = nil
	lst.Unlock()
	return t
}

func (t *udpTransaction) udpMessageForDir(
	header *mcUDPHeader,
	dir applayer.NetDirection,
) *udpMessage {
	udpMsg := t.messages[dir]
	if udpMsg == nil {
		udpMsg = newUDPMessage(header)
		t.messages[dir] = udpMsg
	}
	return udpMsg
}

func newUDPMessage(header *mcUDPHeader) *udpMessage {
	udpMsg := &udpMessage{
		numDatagrams: header.numDatagrams,
		count:        0,
	}
	if header.numDatagrams > 1 {
		udpMsg.datagrams = make([][]byte, header.numDatagrams)
	}
	return udpMsg
}

func (msg *udpMessage) addDatagram(
	header *mcUDPHeader,
	data []byte,
) *streambuf.Buffer {
	if msg.isComplete {
		return nil
	}

	if msg.numDatagrams == 1 {
		msg.isComplete = true
		return streambuf.NewFixed(data)
	}

	if msg.count < msg.numDatagrams {
		if msg.datagrams[header.seqNumber] != nil {
			return nil
		}
		msg.datagrams[header.seqNumber] = data
		msg.count++
	}

	if msg.count < msg.numDatagrams {
		return nil
	}

	buffer := streambuf.New(nil)
	for _, payload := range msg.datagrams {
		buffer.Append(payload)
	}
	msg.isComplete = true
	msg.datagrams = nil
	buffer.Fix()
	return buffer
}

func parseUDPHeader(buf *streambuf.Buffer) (mcUDPHeader, error) {
	var h mcUDPHeader
	h.requestID, _ = buf.ReadNetUint16()
	h.seqNumber, _ = buf.ReadNetUint16()
	h.numDatagrams, _ = buf.ReadNetUint16()
	buf.Advance(2) // ignore reserved
	return h, buf.Err()
}

func parseUDP(
	config *parserConfig,
	ts time.Time,
	buf *streambuf.Buffer,
) (*message, error) {
	parser := newParser(config)
	msg, err := parser.parse(buf, ts)
	if err != nil && msg == nil {
		err = errUDPIncompleteMessage
	}
	return msg, err
}
