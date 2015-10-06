package redis

import (
	"bytes"
	"strconv"
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

const (
	START = iota
	BULK_ARRAY
	SIMPLE_MESSAGE
)

type RedisMessage struct {
	Ts            time.Time
	NumberOfBulks int64
	Bulks         []string

	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
	Direction    uint8

	IsRequest bool
	IsError   bool
	Message   string
	Method    string
	Path      string
	Size      int

	parseState int
	start      int
	end        int
}

type RedisStream struct {
	tcptuple *common.TcpTuple

	data []byte

	parseOffset   int
	bytesReceived int

	message *RedisMessage
}

type RedisTransaction struct {
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

	Request_raw  string
	Response_raw string
}

// Keep sorted for future command addition
var RedisCommands = map[string]struct{}{
	"APPEND":           struct{}{},
	"AUTH":             struct{}{},
	"BGREWRITEAOF":     struct{}{},
	"BGSAVE":           struct{}{},
	"BITCOUNT":         struct{}{},
	"BITOP":            struct{}{},
	"BITPOS":           struct{}{},
	"BLPOP":            struct{}{},
	"BRPOP":            struct{}{},
	"BRPOPLPUSH":       struct{}{},
	"CLIENT GETNAME":   struct{}{},
	"CLIENT KILL":      struct{}{},
	"CLIENT LIST":      struct{}{},
	"CLIENT PAUSE":     struct{}{},
	"CLIENT SETNAME":   struct{}{},
	"CONFIG GET":       struct{}{},
	"CONFIG RESETSTAT": struct{}{},
	"CONFIG REWRITE":   struct{}{},
	"CONFIG SET":       struct{}{},
	"DBSIZE":           struct{}{},
	"DEBUG OBJECT":     struct{}{},
	"DEBUG SEGFAULT":   struct{}{},
	"DECR":             struct{}{},
	"DECRBY":           struct{}{},
	"DEL":              struct{}{},
	"DISCARD":          struct{}{},
	"DUMP":             struct{}{},
	"ECHO":             struct{}{},
	"EVAL":             struct{}{},
	"EVALSHA":          struct{}{},
	"EXEC":             struct{}{},
	"EXISTS":           struct{}{},
	"EXPIRE":           struct{}{},
	"EXPIREAT":         struct{}{},
	"FLUSHALL":         struct{}{},
	"FLUSHDB":          struct{}{},
	"GET":              struct{}{},
	"GETBIT":           struct{}{},
	"GETRANGE":         struct{}{},
	"GETSET":           struct{}{},
	"HDEL":             struct{}{},
	"HEXISTS":          struct{}{},
	"HGET":             struct{}{},
	"HGETALL":          struct{}{},
	"HINCRBY":          struct{}{},
	"HINCRBYFLOAT":     struct{}{},
	"HKEYS":            struct{}{},
	"HLEN":             struct{}{},
	"HMGET":            struct{}{},
	"HMSET":            struct{}{},
	"HSCAN":            struct{}{},
	"HSET":             struct{}{},
	"HSETINX":          struct{}{},
	"HVALS":            struct{}{},
	"INCR":             struct{}{},
	"INCRBY":           struct{}{},
	"INCRBYFLOAT":      struct{}{},
	"INFO":             struct{}{},
	"KEYS":             struct{}{},
	"LASTSAVE":         struct{}{},
	"LINDEX":           struct{}{},
	"LINSERT":          struct{}{},
	"LLEN":             struct{}{},
	"LPOP":             struct{}{},
	"LPUSH":            struct{}{},
	"LPUSHX":           struct{}{},
	"LRANGE":           struct{}{},
	"LREM":             struct{}{},
	"LSET":             struct{}{},
	"LTRIM":            struct{}{},
	"MGET":             struct{}{},
	"MIGRATE":          struct{}{},
	"MONITOR":          struct{}{},
	"MOVE":             struct{}{},
	"MSET":             struct{}{},
	"MSETNX":           struct{}{},
	"MULTI":            struct{}{},
	"OBJECT":           struct{}{},
	"PERSIST":          struct{}{},
	"PEXPIRE":          struct{}{},
	"PEXPIREAT":        struct{}{},
	"PFADD":            struct{}{},
	"PFCOUNT":          struct{}{},
	"PFMERGE":          struct{}{},
	"PING":             struct{}{},
	"PSETEX":           struct{}{},
	"PSUBSCRIBE":       struct{}{},
	"PTTL":             struct{}{},
	"PUBLISH":          struct{}{},
	"PUBSUB":           struct{}{},
	"PUNSUBSCRIBE":     struct{}{},
	"QUIT":             struct{}{},
	"RANDOMKEY":        struct{}{},
	"RENAME":           struct{}{},
	"RENAMENX":         struct{}{},
	"RESTORE":          struct{}{},
	"RPOP":             struct{}{},
	"RPOPLPUSH":        struct{}{},
	"RPUSH":            struct{}{},
	"RPUSHX":           struct{}{},
	"SADD":             struct{}{},
	"SAVE":             struct{}{},
	"SCAN":             struct{}{},
	"SCARD":            struct{}{},
	"SCRIPT EXISTS":    struct{}{},
	"SCRIPT FLUSH":     struct{}{},
	"SCRIPT KILL":      struct{}{},
	"SCRIPT LOAD":      struct{}{},
	"SDIFF":            struct{}{},
	"SDIFFSTORE":       struct{}{},
	"SELECT":           struct{}{},
	"SET":              struct{}{},
	"SETBIT":           struct{}{},
	"SETEX":            struct{}{},
	"SETNX":            struct{}{},
	"SETRANGE":         struct{}{},
	"SHUTDOWN":         struct{}{},
	"SINTER":           struct{}{},
	"SINTERSTORE":      struct{}{},
	"SISMEMBER":        struct{}{},
	"SLAVEOF":          struct{}{},
	"SLOWLOG":          struct{}{},
	"SMEMBERS":         struct{}{},
	"SMOVE":            struct{}{},
	"SORT":             struct{}{},
	"SPOP":             struct{}{},
	"SRANDMEMBER":      struct{}{},
	"SREM":             struct{}{},
	"SSCAN":            struct{}{},
	"STRLEN":           struct{}{},
	"SUBSCRIBE":        struct{}{},
	"SUNION":           struct{}{},
	"SUNIONSTORE":      struct{}{},
	"SYNC":             struct{}{},
	"TIME":             struct{}{},
	"TTL":              struct{}{},
	"TYPE":             struct{}{},
	"UNSUBSCRIBE":      struct{}{},
	"UNWATCH":          struct{}{},
	"WATCH":            struct{}{},
	"ZADD":             struct{}{},
	"ZCARD":            struct{}{},
	"ZCOUNT":           struct{}{},
	"ZINCRBY":          struct{}{},
	"ZINTERSTORE":      struct{}{},
	"ZRANGE":           struct{}{},
	"ZRANGEBYSCORE":    struct{}{},
	"ZRANK":            struct{}{},
	"ZREM":             struct{}{},
	"ZREMRANGEBYLEX":   struct{}{},
	"ZREMRANGEBYRANK":  struct{}{},
	"ZREMRANGEBYSCORE": struct{}{},
	"ZREVRANGE":        struct{}{},
	"ZREVRANGEBYSCORE": struct{}{},
	"ZREVRANK":         struct{}{},
	"ZSCAN":            struct{}{},
	"ZSCORE":           struct{}{},
	"ZUNIONSTORE":      struct{}{},
}

type Redis struct {
	// config
	Ports         []int
	Send_request  bool
	Send_response bool

	transactions *common.Cache

	results publisher.Client
}

func (redis *Redis) getTransaction(k common.HashableTcpTuple) *RedisTransaction {
	v := redis.transactions.Get(k)
	if v != nil {
		return v.(*RedisTransaction)
	}
	return nil
}

func (redis *Redis) InitDefaults() {
	redis.Send_request = false
	redis.Send_response = false
}

func (redis *Redis) setFromConfig(config config.Redis) error {

	redis.Ports = config.Ports

	if config.Send_request != nil {
		redis.Send_request = *config.Send_request
	}
	if config.Send_response != nil {
		redis.Send_response = *config.Send_response
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

	redis.transactions = common.NewCache(protos.DefaultTransactionExpiration,
		protos.DefaultTransactionHashSize)
	redis.transactions.StartJanitor(protos.DefaultTransactionExpiration)
	redis.results = results

	return nil
}

func (stream *RedisStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.parseOffset:]
	stream.parseOffset = 0
	stream.message = nil
	stream.message.Bulks = []string{}
}

func redisMessageParser(s *RedisStream) (bool, bool) {

	var err error
	var value string
	m := s.message

	iserror := false

	for s.parseOffset < len(s.data) {

		if s.data[s.parseOffset] == '*' {
			//Arrays

			m.parseState = BULK_ARRAY
			m.start = s.parseOffset
			logp.Debug("redis", "start %d", m.start)

			found, line, off := readLine(s.data, s.parseOffset)
			if !found {
				logp.Debug("redis", "End of line not found, waiting for more data")
				return true, false
			}
			logp.Debug("redis", "line %s: %d", line, off)

			if len(line) == 3 && line[1] == '-' && line[2] == '1' {
				//Null array
				s.parseOffset = off
				value = "nil"
			} else if len(line) == 2 && line[1] == '0' {
				// Empty array
				s.parseOffset = off
				value = "[]"
			} else {

				m.NumberOfBulks, err = strconv.ParseInt(line[1:], 10, 64)

				if err != nil {
					logp.Err("Failed to read number of bulk messages: %s", err)
					return false, false
				}
				s.parseOffset = off
				m.Bulks = []string{}

				continue
			}

		} else if s.data[s.parseOffset] == '$' {
			// Bulk Strings
			if m.parseState == START {
				m.parseState = SIMPLE_MESSAGE
				m.start = s.parseOffset
			}
			starting_offset := s.parseOffset

			found, line, off := readLine(s.data, s.parseOffset)
			if !found {
				logp.Debug("redis", "End of line not found, waiting for more data")
				s.parseOffset = starting_offset
				return true, false
			}
			logp.Debug("redis", "line %s: %d", line, off)

			if len(line) == 3 && line[1] == '-' && line[2] == '1' {
				// NULL Bulk Reply
				value = "nil"
				s.parseOffset = off
			} else {
				length, err := strconv.ParseInt(line[1:], 10, 64)
				if err != nil {
					logp.Err("Failed to read bulk message: %s", err)
					return false, false
				}

				s.parseOffset = off

				found, line, off = readLine(s.data, s.parseOffset)
				if !found {
					logp.Debug("redis", "End of line not found, waiting for more data")
					s.parseOffset = starting_offset
					return true, false
				}
				logp.Debug("redis", "line %s: %d", line, off)

				if int64(len(line)) != length {
					logp.Err("Wrong length of data: %d instead of %d", len(line), length)
					return false, false
				}
				value = line
				s.parseOffset = off
			}

		} else if s.data[s.parseOffset] == ':' {
			// Integers
			if m.parseState == START {
				// it's not in a bulk message
				m.parseState = SIMPLE_MESSAGE
				m.start = s.parseOffset
			}

			found, line, off := readLine(s.data, s.parseOffset)
			if !found {
				return true, false
			}
			n, err := strconv.ParseInt(line[1:], 10, 64)

			if err != nil {
				logp.Err("Failed to read integer reply: %s", err)
				return false, false
			}
			value = strconv.Itoa(int(n))
			s.parseOffset = off

		} else if s.data[s.parseOffset] == '+' {
			// Simple Strings
			if m.parseState == START {
				// it's not in a bulk message
				m.parseState = SIMPLE_MESSAGE
				m.start = s.parseOffset
			}
			found, line, off := readLine(s.data, s.parseOffset)
			if !found {
				return true, false
			}

			value = line[1:]
			s.parseOffset = off
		} else if s.data[s.parseOffset] == '-' {
			// Errors
			if m.parseState == START {
				// it's not in a bulk message
				m.parseState = SIMPLE_MESSAGE
				m.start = s.parseOffset
			}
			found, line, off := readLine(s.data, s.parseOffset)
			if !found {
				return true, false
			}
			iserror = true

			value = line[1:]
			s.parseOffset = off
		} else {
			logp.Debug("redis", "Unexpected message starting with %s", s.data[s.parseOffset:])
			return false, false
		}

		// add value
		if m.NumberOfBulks > 0 {
			m.NumberOfBulks = m.NumberOfBulks - 1
			m.Bulks = append(m.Bulks, value)

			if len(m.Bulks) == 1 {
				logp.Debug("redis", "Value: %s", value)
				// first word.
				// check if it's a command
				if isRedisCommand(value) {
					logp.Debug("redis", "is request")
					m.IsRequest = true
					m.Method = value
				}
			}

			if len(m.Bulks) == 2 {
				// second word. This is usually the path
				if m.IsRequest {
					m.Path = value
				}
			}

			if m.NumberOfBulks == 0 {
				// the last bulk received
				if m.IsRequest {
					m.Message = strings.Join(m.Bulks, " ")
				} else {
					m.Message = "[" + strings.Join(m.Bulks, ", ") + "]"
				}
				m.end = s.parseOffset
				m.Size = m.end - m.start
				return true, true
			}
		} else {
			m.Message = value
			m.end = s.parseOffset
			m.Size = m.end - m.start
			if iserror {
				m.IsError = true
			}
			return true, true
		}

	} //end for

	return true, false
}

func readLine(data []byte, offset int) (bool, string, int) {
	q := bytes.Index(data[offset:], []byte("\r\n"))
	if q == -1 {
		return false, "", 0
	}
	return true, string(data[offset : offset+q]), offset + q + 2
}

type redisPrivateData struct {
	Data [2]*RedisStream
}

func (redis *Redis) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParseRedis exception")

	priv := redisPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(redisPrivateData)
		if !ok {
			priv = redisPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &RedisStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &RedisMessage{Ts: pkt.Ts},
		}
	} else {
		// concatenate bytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("redis", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	stream := priv.Data[dir]
	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &RedisMessage{Ts: pkt.Ts}
		}

		ok, complete := redisMessageParser(priv.Data[dir])

		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
			logp.Debug("redis", "Ignore Redis message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {

			if stream.message.IsRequest {
				logp.Debug("redis", "REDIS request message: %s", stream.message.Message)
			} else {
				logp.Debug("redis", "REDIS response message: %s", stream.message.Message)
			}

			// all ok, go to next level
			redis.handleRedis(stream.message, tcptuple, dir)

			// and reset message
			stream.PrepareForNewMessage()
		} else {
			// wait for more data
			break
		}
	}

	return priv
}

func isRedisCommand(key string) bool {
	_, exists := RedisCommands[strings.ToUpper(key)]
	return exists
}

func (redis *Redis) handleRedis(m *RedisMessage, tcptuple *common.TcpTuple,
	dir uint8) {

	m.TcpTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

	if m.IsRequest {
		redis.receivedRedisRequest(m)
	} else {
		redis.receivedRedisResponse(m)
	}
}

func (redis *Redis) receivedRedisRequest(msg *RedisMessage) {
	tuple := msg.TcpTuple
	trans := redis.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.Redis != nil {
			logp.Warn("Two requests without a Response. Dropping old request")
		}
	} else {
		trans = &RedisTransaction{Type: "redis", tuple: tuple}
		redis.transactions.Put(tuple.Hashable(), trans)
	}

	trans.Redis = common.MapStr{}
	trans.Method = msg.Method
	trans.Path = msg.Path
	trans.Query = msg.Message
	trans.Request_raw = msg.Message
	trans.BytesIn = msg.Size

	trans.cmdline = msg.CmdlineTuple
	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000) // transactions have microseconds resolution
	trans.JsTs = msg.Ts
	trans.Src = common.Endpoint{
		Ip:   msg.TcpTuple.Src_ip.String(),
		Port: msg.TcpTuple.Src_port,
		Proc: string(msg.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   msg.TcpTuple.Dst_ip.String(),
		Port: msg.TcpTuple.Dst_port,
		Proc: string(msg.CmdlineTuple.Dst),
	}
	if msg.Direction == tcp.TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}
}

func (redis *Redis) receivedRedisResponse(msg *RedisMessage) {
	tuple := msg.TcpTuple
	trans := redis.getTransaction(tuple.Hashable())
	if trans == nil {
		logp.Warn("Response from unknown transaction. Ignoring.")
		return
	}
	// check if the request was received
	if trans.Redis == nil {
		logp.Warn("Response from unknown transaction. Ignoring.")
		return

	}

	trans.IsError = msg.IsError
	if msg.IsError {
		trans.Redis["error"] = msg.Message
	} else {
		trans.Redis["return_value"] = msg.Message
	}

	trans.BytesOut = msg.Size
	trans.Response_raw = msg.Message

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

	redis.publishTransaction(trans)
	redis.transactions.Delete(trans.tuple.Hashable())

	logp.Debug("redis", "Redis transaction completed: %s", trans.Redis)
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

func (redis *Redis) publishTransaction(t *RedisTransaction) {

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
	if redis.Send_request {
		event["request"] = t.Request_raw
	}
	if redis.Send_response {
		event["response"] = t.Response_raw
	}
	event["redis"] = common.MapStr(t.Redis)
	event["method"] = strings.ToUpper(t.Method)
	event["resource"] = t.Path
	event["query"] = t.Query
	event["bytes_in"] = uint64(t.BytesIn)
	event["bytes_out"] = uint64(t.BytesOut)

	event["timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	redis.results.PublishEvent(event)
}
