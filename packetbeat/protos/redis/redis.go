package redis

import (
	"bytes"
	"expvar"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/applayer"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

type stream struct {
	applayer.Stream
	parser   parser
	tcptuple *common.TcpTuple
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

	results publish.Transactions
}

var (
	debugf  = logp.MakeDebug("redis")
	isDebug = false
)

var (
	unmatchedResponses = expvar.NewInt("redis.unmatched_responses")
)

func init() {
	protos.Register("redis", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Redis{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func (redis *Redis) init(results publish.Transactions, config *redisConfig) error {
	redis.setFromConfig(config)

	redis.results = results
	isDebug = logp.IsDebug("redis")

	return nil
}

func (redis *Redis) setFromConfig(config *redisConfig) {
	redis.Ports = config.Ports
	redis.SendRequest = config.SendRequest
	redis.SendResponse = config.SendResponse
	redis.transactionTimeout = config.TransactionTimeout
}

func (redis *Redis) GetPorts() []int {
	return redis.Ports
}

func (s *stream) PrepareForNewMessage() {
	parser := &s.parser
	s.Stream.Reset()
	parser.reset()
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
		st = newStream(pkt.Ts, tcptuple)
		conn.Streams[dir] = st
		if isDebug {
			debugf("new stream: %p (dir=%v, len=%v)", st, dir, len(pkt.Payload))
		}
	}

	if err := st.Append(pkt.Payload); err != nil {
		if isDebug {
			debugf("%v, dropping TCP stream: ", err)
		}
		return nil
	}
	if isDebug {
		debugf("stream add data: %p (dir=%v, len=%v)", st, dir, len(pkt.Payload))
	}

	for st.Buf.Len() > 0 {
		if st.parser.message == nil {
			st.parser.message = newMessage(pkt.Ts)
		}

		ok, complete := st.parser.parse(&st.Buf)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			conn.Streams[dir] = nil
			if isDebug {
				debugf("Ignore Redis message. Drop tcp stream. Try parsing with the next segment")
			}
			return conn
		}

		if !complete {
			// wait for more data
			break
		}

		msg := st.parser.message
		if isDebug {
			if msg.IsRequest {
				debugf("REDIS (%p) request message: %s", conn, msg.Message)
			} else {
				debugf("REDIS (%p) response message: %s", conn, msg.Message)
			}
		}

		// all ok, go to next level and reset stream for new message
		redis.handleRedis(conn, msg, tcptuple, dir)
		st.PrepareForNewMessage()
	}

	return conn
}

func newStream(ts time.Time, tcptuple *common.TcpTuple) *stream {
	s := &stream{
		tcptuple: tcptuple,
	}
	s.parser.message = newMessage(ts)
	s.Stream.Init(tcp.TCP_MAX_DATA_IN_STREAM)
	return s
}

func newMessage(ts time.Time) *redisMessage {
	return &redisMessage{Ts: ts}
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
			debugf("Response from unknown transaction. Ignoring")
			unmatchedResponses.Add(1)
			conn.responses.pop()
		}
		return
	}

	// merge requests with responses into transactions
	for !conn.responses.empty() && !conn.requests.empty() {
		requ := conn.requests.pop()
		resp := conn.responses.pop()

		if redis.results != nil {
			event := redis.newTransaction(requ, resp)
			redis.results.PublishTransaction(event)
		}
	}
}

func (redis *Redis) newTransaction(requ, resp *redisMessage) common.MapStr {
	error := common.OK_STATUS
	if resp.IsError {
		error = common.ERROR_STATUS
	}

	var returnValue map[string]common.NetString
	if resp.IsError {
		returnValue = map[string]common.NetString{
			"error": resp.Message,
		}
	} else {
		returnValue = map[string]common.NetString{
			"return_value": resp.Message,
		}
	}

	src := &common.Endpoint{
		Ip:   requ.TcpTuple.Src_ip.String(),
		Port: requ.TcpTuple.Src_port,
		Proc: string(requ.CmdlineTuple.Src),
	}
	dst := &common.Endpoint{
		Ip:   requ.TcpTuple.Dst_ip.String(),
		Port: requ.TcpTuple.Dst_port,
		Proc: string(requ.CmdlineTuple.Dst),
	}
	if requ.Direction == tcp.TcpDirectionReverse {
		src, dst = dst, src
	}

	// resp_time in milliseconds
	responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

	event := common.MapStr{
		"@timestamp":   common.Time(requ.Ts),
		"type":         "redis",
		"status":       error,
		"responsetime": responseTime,
		"redis":        returnValue,
		"method":       common.NetString(bytes.ToUpper(requ.Method)),
		"resource":     requ.Path,
		"query":        requ.Message,
		"bytes_in":     uint64(requ.Size),
		"bytes_out":    uint64(resp.Size),
		"src":          src,
		"dst":          dst,
	}
	if redis.SendRequest {
		event["request"] = requ.Message
	}
	if redis.SendResponse {
		event["response"] = resp.Message
	}

	return event
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
	return msg
}

func (ml *messageList) last() *redisMessage {
	return ml.tail
}
