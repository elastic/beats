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

// Keep sorted for future command addition
var RedisCommands = map[string]struct{}{
    "APPEND":           struct{}{},
    "AUTH":             struct{}{},
    "BGREWRITEAOF":     struct{}{},
    "BGSAVE":           struct{}{},
    "BITCOUNT":         struct{}{},
    "BITOP":            struct{}{},
    "BLPOP":            struct{}{},
    "BRPOP":            struct{}{},
    "BRPOPLPUSH":       struct{}{},
    "CLIENT GETNAME":   struct{}{},
    "CLIENT KILL":      struct{}{},
    "CLIENT LIST":      struct{}{},
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
    "ZREMRANGEBYRANK":  struct{}{},
    "ZREMRANGEBYSCORE": struct{}{},
    "ZREVRANGE":        struct{}{},
    "ZREVRANGEBYSCORE": struct{}{},
    "ZREVRANK":         struct{}{},
    "ZSCAN":            struct{}{},
    "ZSCORE":           struct{}{},
    "ZUNIONSTORE":      struct{}{},
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
    defer RECOVER("ParseRedis exception")

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
    _, exists := RedisCommands[key]
    return exists
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
        Ip:   Ipv4_Ntoa(tuple.Src_ip),
        Port: tuple.Src_port,
        Proc: string(msg.CmdlineTuple.Src),
    }
    trans.Dst = DbEndpoint{
        Ip:   Ipv4_Ntoa(tuple.Dst_ip),
        Port: tuple.Dst_port,
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

    trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

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
