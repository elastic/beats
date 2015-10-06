package memcache

// Memcache UDP Protocol Plugin implementation.

import (
	"sync"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/common/streambuf"
	"github.com/elastic/libbeat/logp"

	"github.com/elastic/packetbeat/procs"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/applayer"
)

type udpMemcache struct {
	udpConfig      udpConfig
	udpConnections map[common.HashableIpPortTuple]*udpConnection
	udpExpTrans    udpExpTransList
}

type udpConfig struct {
	transTimeout time.Duration
}

type mcUdpHeader struct {
	requestId    uint16
	seqNumber    uint16
	numDatagrams uint16
}

type udpConnection struct {
	tuple        common.IpPortTuple
	transactions map[uint16]*udpTransaction
	memcache     *Memcache
}

type udpTransaction struct {
	requestId uint16
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

const defaultUdpTransDuration uint = 200

func (mc *Memcache) ParseUdp(pkt *protos.Packet) {
	defer logp.Recover("ParseMemcache(UDP) exception")

	buffer := streambuf.NewFixed(pkt.Payload)
	header, err := parseUdpHeader(buffer)
	if err != nil {
		debug("parsing memcache udp header failed")
		return
	}

	debug("new udp datagram requestId=%v, seqNumber=%v, numDatagrams=%v",
		header.requestId, header.seqNumber, header.numDatagrams)

	// find connection object based on ips and ports (forward->reverse connection)
	connection, dir := mc.getUdpConnection(&pkt.Tuple)
	debug("udp connection: %p", connection)

	// get udp transaction combining forward/reverse direction 'streams'
	// for current requestId
	trans := connection.udpTransactionForId(header.requestId)
	debug("udp transaction (id=%v): %p", header.requestId, trans)

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
		msg, err := parseUdp(&mc.config, pkt.Ts, payload)
		if err != nil {
			logp.Warn("failed to parse memcached(UDP) message: %s", err)
			connection.killTransaction(trans)
			return
		}

		// apply memcached to transaction
		done, err = mc.onUdpMessage(trans, &pkt.Tuple, dir, msg)
		if err != nil {
			logp.Warn("error processing memcache message: %s", err)
			connection.killTransaction(trans)
			done = true
		}
	}
	if !done {
		trans.timer = time.AfterFunc(mc.udpConfig.transTimeout, func() {
			debug("transaction timeout -> forward")
			mc.onUdpTrans(trans)
			mc.udpExpTrans.push(trans)
		})
	}
}

func (mc *Memcache) getUdpConnection(
	tuple *common.IpPortTuple,
) (*udpConnection, applayer.NetDirection) {
	connection := mc.udpConnections[tuple.Hashable()]
	if connection != nil {
		return connection, applayer.NetOriginalDirection
	}
	connection = mc.udpConnections[tuple.RevHashable()]
	if connection != nil {
		return connection, applayer.NetReverseDirection
	}

	connection = newUdpConnection(mc, tuple)
	mc.udpConnections[tuple.Hashable()] = connection
	return connection, applayer.NetOriginalDirection
}

func (mc *Memcache) onUdpMessage(
	trans *udpTransaction,
	tuple *common.IpPortTuple,
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
	msg.Transport = applayer.TransportUdp
	msg.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tuple)

	done := false
	var err error
	if msg.IsRequest {
		msg.isComplete = true
		trans.request = msg
		waitResponse := msg.noreply ||
			(!msg.isBinary && msg.command.code != MemcacheCmdQuit) ||
			(msg.isBinary && msg.opcode != opcodeQuitQ)
		done = !waitResponse
	} else {
		msg.isComplete = true
		trans.response = msg
	}

	done = done || (trans.request != nil && trans.response != nil)
	if done {
		err = mc.onUdpTrans(trans)
		trans.connection.killTransaction(trans)
	}
	return done, err
}

func (mc *Memcache) onUdpTrans(udp *udpTransaction) error {

	debug("received memcache(udp) transaction")
	trans := newTransaction(udp.request, udp.response)
	return mc.finishTransaction(trans)
}

func newUdpConnection(mc *Memcache, tuple *common.IpPortTuple) *udpConnection {
	c := &udpConnection{
		tuple:        *tuple,
		memcache:     mc,
		transactions: make(map[uint16]*udpTransaction),
	}
	return c
}

func (c *udpConnection) udpTransactionForId(requestId uint16) *udpTransaction {
	trans := c.transactions[requestId]
	if trans != nil && trans.timer != nil {
		stopped := trans.timer.Stop()
		if !stopped {
			logp.Warn("timer stopped while processing transaction -> create new transaction")
			trans = nil
		}
	}
	if trans == nil {
		trans = &udpTransaction{
			requestId:  requestId,
			connection: c,
		}
		c.transactions[requestId] = trans
	} else {
		trans.timer = nil
	}

	return trans
}

func (c *udpConnection) killTransaction(t *udpTransaction) {
	if t.timer != nil {
		t.timer.Stop()
	}

	if c.transactions[t.requestId] != t {
		// transaction was already replaced
		return
	}

	delete(c.transactions, t.requestId)
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
	header *mcUdpHeader,
	dir applayer.NetDirection,
) *udpMessage {
	udpMsg := t.messages[dir]
	if udpMsg == nil {
		udpMsg = newUdpMessage(header)
		t.messages[dir] = udpMsg
	}
	return udpMsg
}

func newUdpMessage(header *mcUdpHeader) *udpMessage {
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
	header *mcUdpHeader,
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

func parseUdpHeader(buf *streambuf.Buffer) (mcUdpHeader, error) {
	var h mcUdpHeader
	h.requestId, _ = buf.ReadNetUint16()
	h.seqNumber, _ = buf.ReadNetUint16()
	h.numDatagrams, _ = buf.ReadNetUint16()
	buf.Advance(2) // ignore reserved
	return h, buf.Err()
}

func parseUdp(
	config *parserConfig,
	ts time.Time,
	buf *streambuf.Buffer,
) (*message, error) {
	parser := newParser(config)
	msg, err := parser.parse(buf, ts)
	if err != nil && msg == nil {
		err = ErrUdpIncompleteMessage
	}
	return msg, err
}
