package redis

import (
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"

	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/procs"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/tcp"
)

type stream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset   int
	bytesReceived int

	message *redisMessage
}

type transaction struct {
	Type         string
	tuple        common.TcpTuple
	Src          common.Endpoint
	Dst          common.Endpoint
	ResponseTime int32
	Ts           int64
	JsTs         time.Time
	ts           time.Time
	cmdline      *common.CmdlineTuple
	Method       string
	Path         string
	Query        string
	IsError      bool
	BytesOut     int
	BytesIn      int

	Redis common.MapStr

	RequestRaw  string
	ResponseRaw string
}

type redisConnectionData struct {
	Streams   [2]*stream
	requests  messageList
	responses messageList
}

type messageList struct {
	head, tail *redisMessage
}

// Redis protocol plugin
type Redis struct {
	// config
	Ports        []int
	SendRequest  bool
	SendResponse bool

	transactionTimeout time.Duration

	results publisher.Client
}

var debug = logp.MakeDebug("redis")

func (redis *Redis) InitDefaults() {
	redis.SendRequest = false
	redis.SendResponse = false
	redis.transactionTimeout = protos.DefaultTransactionExpiration
}

func (redis *Redis) setFromConfig(config config.Redis) error {
	redis.Ports = config.Ports

	if config.SendRequest != nil {
		redis.SendRequest = *config.SendRequest
	}
	if config.SendResponse != nil {
		redis.SendResponse = *config.SendResponse
	}
	if config.TransactionTimeout != nil && *config.TransactionTimeout > 0 {
		redis.transactionTimeout = time.Duration(*config.TransactionTimeout) * time.Second
	}
	return nil
}

func (redis *Redis) GetPorts() []int {
	return redis.Ports
}

func (redis *Redis) Init(test_mode bool, results publisher.Client) error {
	redis.InitDefaults()
	if !test_mode {
		redis.setFromConfig(config.ConfigSingleton.Protocols.Redis)
	}

	redis.results = results

	return nil
}

func (s *stream) PrepareForNewMessage() {
	s.data = s.data[s.parseOffset:]
	s.parseOffset = 0
	s.message = nil
}

func (redis *Redis) ConnectionTimeout() time.Duration {
	return redis.transactionTimeout
}

func (redis *Redis) Parse(
	pkt *protos.Packet,
	tcptuple *common.TcpTuple,
	dir uint8,
	private protos.ProtocolData,
) protos.ProtocolData {
	defer logp.Recover("ParseRedis exception")

	conn := ensureRedisConnection(private)
	conn = redis.doParse(conn, pkt, tcptuple, dir)
	if conn == nil {
		return nil
	}
	return conn
}

func ensureRedisConnection(private protos.ProtocolData) *redisConnectionData {
	if private == nil {
		return &redisConnectionData{}
	}

	priv, ok := private.(*redisConnectionData)
	if !ok {
		logp.Warn("redis connection data type error, create new one")
		return &redisConnectionData{}
	}
	if priv == nil {
		logp.Warn("Unexpected: redis connection data not set, create new one")
		return &redisConnectionData{}
	}

	return priv
}

func (redis *Redis) doParse(
	conn *redisConnectionData,
	pkt *protos.Packet,
	tcptuple *common.TcpTuple,
	dir uint8,
) *redisConnectionData {

	st := conn.Streams[dir]
	if st == nil {
		st = &stream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  newMessage(pkt.Ts),
		}
		conn.Streams[dir] = st
	} else {
		st.data = append(st.data, pkt.Payload...)
		if len(st.data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("redis", "Stream data too large, dropping TCP stream")
			conn.Streams[dir] = nil
		}
		return conn
	}

	for len(st.data) > 0 {
		if st.message == nil {
			st.message = newMessage(pkt.Ts)
		}

		ok, complete := redisMessageParser(st)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.Streams[dir] = nil
			debug("Ignore Redis message. Drop tcp stream. Try parsing with the next segment")
			return conn
		}

		if !complete {
			// wait for more data
			break
		}

		if st.message.IsRequest {
			debug("REDIS (%p) request message: %s", conn, st.message.Message)
		} else {
			debug("REDIS (%p) response message: %s", conn, st.message.Message)
		}

		// all ok, go to next level and reset stream for new message
		redis.handleRedis(conn, st.message, tcptuple, dir)
		st.PrepareForNewMessage()
	}

	return conn
}

func newMessage(ts time.Time) *redisMessage {
	return &redisMessage{Ts: ts, Bulks: []string{}}
}

func (redis *Redis) handleRedis(
	conn *redisConnectionData,
	m *redisMessage,
	tcptuple *common.TcpTuple,
	dir uint8,
) {
	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

	if m.IsRequest {
		conn.requests.append(m) // wait for response
	} else {
		conn.responses.append(m)
		redis.correlate(conn)
	}
}

func (redis *Redis) correlate(conn *redisConnectionData) {
	// drop responses with missing requests
	if conn.requests.empty() {
		for !conn.responses.empty() {
			logp.Warn("Response from unknown transaction. Ignoring")
			conn.responses.pop()
		}
		return
	}

	// merge requests with responses into transactions
	for !conn.responses.empty() && !conn.requests.empty() {
		requ := conn.requests.pop()
		resp := conn.responses.pop()
		trans := newTransaction(requ, resp)

		debug("REDIS (%p) transaction completed: %s", conn, trans.Redis)
		redis.publishTransaction(trans)
	}
}

func newTransaction(requ, resp *redisMessage) *transaction {
	trans := &transaction{Type: "redis", tuple: requ.TcpTuple}

	// init from request
	trans.Redis = common.MapStr{}
	trans.Method = requ.Method
	trans.Path = requ.Path
	trans.Query = requ.Message
	trans.RequestRaw = requ.Message
	trans.BytesIn = requ.Size

	trans.cmdline = requ.CmdlineTuple
	trans.ts = requ.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000) // transactions have microseconds resolution
	trans.JsTs = requ.Ts
	trans.Src = common.Endpoint{
		Ip:   requ.TcpTuple.Src_ip.String(),
		Port: requ.TcpTuple.Src_port,
		Proc: string(requ.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   requ.TcpTuple.Dst_ip.String(),
		Port: requ.TcpTuple.Dst_port,
		Proc: string(requ.CmdlineTuple.Dst),
	}
	if requ.Direction == tcp.TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}

	// init from response
	trans.IsError = resp.IsError
	if resp.IsError {
		trans.Redis["error"] = resp.Message
	} else {
		trans.Redis["return_value"] = resp.Message
	}

	trans.BytesOut = resp.Size
	trans.ResponseRaw = resp.Message

	trans.ResponseTime = int32(resp.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	return trans
}

func (redis *Redis) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	// tsg: being packet loss tolerant is probably not very useful for Redis,
	// because most requests/response tend to fit in a single packet.

	return private, true
}

func (redis *Redis) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO: check if we have pending data that we can send up the stack

	return private
}

func (redis *Redis) publishTransaction(t *transaction) {
	if redis.results == nil {
		return
	}

	event := common.MapStr{}
	event["type"] = "redis"
	if !t.IsError {
		event["status"] = common.OK_STATUS
	} else {
		event["status"] = common.ERROR_STATUS
	}
	event["responsetime"] = t.ResponseTime
	if redis.SendRequest {
		event["request"] = t.RequestRaw
	}
	if redis.SendResponse {
		event["response"] = t.ResponseRaw
	}
	event["redis"] = common.MapStr(t.Redis)
	event["method"] = strings.ToUpper(t.Method)
	event["resource"] = t.Path
	event["query"] = t.Query
	event["bytes_in"] = uint64(t.BytesIn)
	event["bytes_out"] = uint64(t.BytesOut)

	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	redis.results.PublishEvent(event)
}

func (ml *messageList) append(msg *redisMessage) {
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

func (ml *messageList) pop() *redisMessage {
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

func (ml *messageList) last() *redisMessage {
	return ml.tail
}
