package main

import (
    "bytes"
    "strconv"
    "strings"
    "time"

    "labix.org/v2/mgo/bson"
)

type RedisMessage struct {
    Ts            time.Time
    NumberOfBulks int64
    Bulks         []string

    Stream_id    uint32
    Tuple        *IpPortTuple
    CmdlineTuple *CmdlineTuple
    Direction    uint8

    IsRequest bool
    Message   string
}

type RedisStream struct {
    tcpStream *TcpStream

    data []byte

    parseOffset   int
    parseState    int
    bytesReceived int

    message *RedisMessage
}

type RedisTransaction struct {
    Type         string
    tuple        TcpTuple
    Src          DbEndpoint
    Dst          DbEndpoint
    ResponseTime int32
    Ts           int64
    JsTs         time.Time
    ts           time.Time
    cmdline      *CmdlineTuple

    Redis bson.M

    Request_raw  string
    Response_raw string

    timer *time.Timer
}

var redisTransactionsMap = make(map[TcpTuple]*RedisTransaction, TransactionsHashSize)

func (stream *RedisStream) PrepareForNewMessage() {
    stream.message.NumberOfBulks = 0
    stream.message.Bulks = []string{}
    stream.message.IsRequest = false
}

func redisMessageParser(s *RedisStream) (bool, bool) {

    var err error
    var value string
    m := s.message

    for s.parseOffset < len(s.data) {

        if s.data[s.parseOffset] == '*' {
            //Multi Bulk Message

            line, off := readLine(s.data, s.parseOffset)

            if len(line) == 3 && line[1] == '-' && line[2] == '1' {
                //NULL Multi Bulk
                s.parseOffset = off
                value = "nil"
            } else {

                m.NumberOfBulks, err = strconv.ParseInt(line[1:], 10, 64)

                if err != nil {
                    ERR("Failed to read number of bulk messages: %s", err)
                    return false, false
                }
                s.parseOffset = off
                m.Bulks = []string{}

                continue
            }

        } else if s.data[s.parseOffset] == '$' {
            // Bulk Reply

            line, off := readLine(s.data, s.parseOffset)

            if len(line) == 3 && line[1] == '-' && line[2] == '1' {
                // NULL Bulk Reply
                value = "nil"
                s.parseOffset = off
            } else {
                length, err := strconv.ParseInt(line[1:], 10, 64)
                if err != nil {
                    ERR("Failed to read bulk message: %s", err)
                    return false, false
                }

                s.parseOffset = off

                line, off = readLine(s.data, s.parseOffset)

                if int64(len(line)) != length {
                    ERR("Wrong length of data: %d instead of %d", len(line), length)
                    return false, false
                }
                value = line
                s.parseOffset = off
            }

        } else if s.data[s.parseOffset] == ':' {
            // Integer reply
            line, off := readLine(s.data, s.parseOffset)
            n, err := strconv.ParseInt(line[1:], 10, 64)

            if err != nil {
                ERR("Failed to read integer reply: %s", err)
                return false, false
            }
            value = string(n)
            s.parseOffset = off

        } else if s.data[s.parseOffset] == '+' {
            // Status Reply
            line, off := readLine(s.data, s.parseOffset)

            value = line[1:]
            s.parseOffset = off
        } else if s.data[s.parseOffset] == '-' {
            // Error Reply
            line, off := readLine(s.data, s.parseOffset)

            value = line[1:]
            s.parseOffset = off
        } else {
            DEBUG("redis", "Unexpected message starting with %s", s.data[s.parseOffset:])
            return false, false
        }

        // add value
        if m.NumberOfBulks > 0 {
            m.NumberOfBulks = m.NumberOfBulks - 1
            m.Bulks = append(m.Bulks, value)

            if len(m.Bulks) == 1 {
                // check if it's a command
                if isRedisCommand(value) {
                    m.IsRequest = true
                }
            }

            if m.NumberOfBulks == 0 {
                // the last bulk received
                m.Message = strings.Join(m.Bulks, " ")
                return true, true
            }
        } else {
            m.Message = value
            return true, true
        }

    }   //end for

    return true, false
}

func readLine(data []byte, offset int) (string, int) {
    q := bytes.Index(data[offset:], []byte("\r\n"))
    return string(data[offset : offset+q]), offset + q + 2
}

func ParseRedis(pkt *Packet, tcp *TcpStream, dir uint8) {

    if tcp.redisData[dir] == nil {
        tcp.redisData[dir] = &RedisStream{
            tcpStream: tcp,
            data:      pkt.payload,
            message:   &RedisMessage{Ts: pkt.ts},
        }
    } else {
        // concatenate bytes
        tcp.redisData[dir].data = append(tcp.redisData[dir].data, pkt.payload...)
    }

    stream := tcp.redisData[dir]
    if stream.message == nil {
        stream.message = &RedisMessage{Ts: pkt.ts}
    }

    ok, complete := redisMessageParser(tcp.redisData[dir])
    if !ok {
        // drop this tcp stream. Will retry parsing with the next
        // segment in it
        tcp.redisData[dir] = nil
        return
    }

    if !ok {
        // drop this tcp stream. Will retry parsing with the next
        // segment in it
        tcp.redisData[dir] = nil
        return
    }

    if complete {

        if stream.message.IsRequest {
            DEBUG("redis", "REDIS request message: %s", stream.message.Message)
        } else {
            DEBUG("redis", "REDIS response message: %s", stream.message.Message)
        }

        // all ok, go to next level
        handleRedis(stream.message, tcp, dir)

        // and reset message
        stream.PrepareForNewMessage()
    }

}

func isRedisCommand(key string) bool {

    RedisCommands := map[string]int{
        "APPEND":           1,
        "AUTH":             1,
        "BGREWRITEAOF":     1,
        "BGSAVE":           1,
        "BITCOUNT":         1,
        "BITOP":            1,
        "BLPOP":            1,
        "BRPOP":            1,
        "BRPOPLPUSH":       1,
        "CLIENT KILL":      1,
        "CLIENT LIST":      1,
        "CLIENT GETNAME":   1,
        "CLIENT SETNAME":   1,
        "CONFIG GET":       1,
        "CONFIG REWRITE":   1,
        "CONFIG SET":       1,
        "CONFIG REsETSTAT": 1,
        "DBSIZE":           1,
        "DEBUG OBJECT":     1,
        "DEBUG SEGFAULT":   1,
        "DECR":             1,
        "DECRBY":           1,
        "DEL":              1,
        "DISCARD":          1,
        "DUMP":             1,
        "ECHO":             1,
        "EVAL":             1,
        "EVALSHA":          1,
        "EXEC":             1,
        "EXISTS":           1,
        "EXPIRE":           1,
        "EXPIREAT":         1,
        "FLUSHALL":         1,
        "GET":              1,
        "GETBIT":           1,
        "GETRANGE":         1,
        "GETSET":           1,
        "HDEL":             1,
        "HEXISTS":          1,
        "HGET":             1,
        "HGETALL":          1,
        "HINCRBY":          1,
        "HINCRBYFLOAT":     1,
        "HKEYS":            1,
        "HLEN":             1,
        "HMGET":            1,
        "HMSET":            1,
        "HSET":             1,
        "HSETINX":          1,
        "HVALS":            1,
        "INCR":             1,
        "INCRBY":           1,
        "INCRBYFLOAT":      1,
        "INFO":             1,
        "KEYS":             1,
        "LASTSAVE":         1,
        "LINDEX":           1,
        "LINSERT":          1,
        "LLEN":             1,
        "LPOP":             1,
        "LPUSH":            1,
        "LPUSHX":           1,
        "LRANGE":           1,
        "LREM":             1,
        "LSET":             1,
        "LTRIM":            1,
        "MGET":             1,
        "MIGRATE":          1,
        "MONITOR":          1,
        "MOVE":             1,
        "MSET":             1,
        "MSETNX":           1,
        "MULTI":            1,
        "OBJECT":           1,
        "PERSIST":          1,
        "PEXPIRE":          1,
        "PEXPIREAT":        1,
        "PING":             1,
        "PSETEX":           1,
        "PSUBSCRIBE":       1,
        "PUBSUB":           1,
        "PTTL":             1,
        "PUBLISH":          1,
        "PUNSUBSCRIBE":     1,
        "QUIT":             1,
        "RANDOMKEY":        1,
        "RENAME":           1,
        "RENAMENX":         1,
        "RESTORE":          1,
        "RPOP":             1,
        "RPOPLPUSH":        1,
        "RPUSH":            1,
        "RPUSHX":           1,
        "SADD":             1,
        "SAVE":             1,
        "SCARD":            1,
        "SCRIPT EXISTS":    1,
        "SCRIPT FLUSH":     1,
        "SCRIPT KILL":      1,
        "SCRIPT LOAD":      1,
        "SDIFF":            1,
        "SDIFFSTORE":       1,
        "SELECT":           1,
        "SET":              1,
        "SETBIT":           1,
        "SETEX":            1,
        "SETNX":            1,
        "SETRANGE":         1,
        "SHUTDOWN":         1,
        "SINTER":           1,
        "SINTERSTORE":      1,
        "SISMEMBER":        1,
        "SLAVEOF":          1,
        "SLOWLOG":          1,
        "SMEMBERS":         1,
        "SMOVE":            1,
        "SORT":             1,
        "SPOP":             1,
        "SRANDMEMBER":      1,
        "SREM":             1,
        "STRLEN":           1,
        "SUBSCRIBE":        1,
        "SUNION":           1,
        "SUNIONSTORE":      1,
        "SYNC":             1,
        "TIME":             1,
        "TTL":              1,
        "TYPE":             1,
        "UNSUBSCRIBE":      1,
        "UNWATCH":          1,
        "WATCH":            1,
        "ZADD":             1,
        "ZCARD":            1,
        "ZCOUNT":           1,
        "ZINCRBY":          1,
        "ZINTERSTORE":      1,
        "ZRANGE":           1,
        "ZRANGEBYSCORE":    1,
        "ZRANK":            1,
        "ZREM":             1,
        "ZREMRANGEBYRANK":  1,
        "ZREMRANGEBYSCORE": 1,
        "ZREVRANGE":        1,
        "ZREVRANGEBYSCORE": 1,
        "ZREVRANK":         1,
        "ZSCORE":           1,
        "ZUNIONSTORE":      1,
        "SCAN":             1,
        "SSCAN":            1,
        "HSCAN":            1,
        "ZSCAN":            1,
    }

    if RedisCommands[key] > 0 {
        return true
    }
    return false
}

func handleRedis(m *RedisMessage, tcp *TcpStream,
    dir uint8) {

    m.Stream_id = tcp.id
    m.Tuple = tcp.tuple
    m.Direction = dir
    m.CmdlineTuple = procWatcher.FindProcessesTuple(tcp.tuple)

    if m.IsRequest {
        receivedRedisRequest(m)
    } else {
        receivedRedisResponse(m)
    }
}

func receivedRedisRequest(msg *RedisMessage) {
    // Add it to the HT
    tuple := TcpTuple{
        Src_ip:    msg.Tuple.Src_ip,
        Dst_ip:    msg.Tuple.Dst_ip,
        Src_port:  msg.Tuple.Src_port,
        Dst_port:  msg.Tuple.Dst_port,
        stream_id: msg.Stream_id,
    }

    trans := redisTransactionsMap[tuple]
    if trans != nil {
        if len(trans.Redis) != 0 {
            WARN("Two requests without a Response. Dropping old request")
        }
    } else {
        trans = &RedisTransaction{Type: "redis", tuple: tuple}
        redisTransactionsMap[tuple] = trans
    }

    var redis bson.M

    DEBUG("redis", "Receive request: %s", redis)

    trans.Redis = bson.M{
        "request": msg.Message,
    }
    trans.Request_raw = msg.Message

    trans.cmdline = msg.CmdlineTuple
    trans.ts = msg.Ts
    trans.Ts = int64(trans.ts.UnixNano() / 1000) // transactions have microseconds resolution
    trans.JsTs = msg.Ts
    trans.Src = DbEndpoint{
        Ip:     Ipv4_Ntoa(tuple.Src_ip),
        Port:   tuple.Src_port,
        Proc: string(msg.CmdlineTuple.Src),
    }
    trans.Dst = DbEndpoint{
        Ip:     Ipv4_Ntoa(tuple.Dst_ip),
        Port:   tuple.Dst_port,
        Proc: string(msg.CmdlineTuple.Dst),
    }

    if trans.timer != nil {
        trans.timer.Stop()
    }
    trans.timer = time.AfterFunc(TransactionTimeout, func() { trans.Expire() })

}

func (trans *RedisTransaction) Expire() {

    // remove from map
    delete(redisTransactionsMap, trans.tuple)
}

func receivedRedisResponse(msg *RedisMessage) {

    tuple := TcpTuple{
        Src_ip:    msg.Tuple.Src_ip,
        Dst_ip:    msg.Tuple.Dst_ip,
        Src_port:  msg.Tuple.Src_port,
        Dst_port:  msg.Tuple.Dst_port,
        stream_id: msg.Stream_id,
    }
    trans := redisTransactionsMap[tuple]
    if trans == nil {
        WARN("Response from unknown transaction. Ignoring.")
        return
    }
    // check if the request was received
    if len(trans.Redis) == 0 {
        WARN("Response from unknown transaction. Ignoring.")
        return

    }

    var redis bson.M

    DEBUG("redis", "Receive response: %s", redis)

    trans.Redis["response"] = msg.Message

    trans.Response_raw = msg.Message

    trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e3) // resp_time in micros

    err := Publisher.PublishRedisTransaction(trans)
    if err != nil {
        WARN("Publish failure: %s", err)
    }

    DEBUG("redis", "Redis transaction completed: %s", trans.Redis)

    // remove from map
    delete(redisTransactionsMap, trans.tuple)
    if trans.timer != nil {
        trans.timer.Stop()
    }

}
