package redis

import (
	"strconv"
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/common/streambuf"
	"github.com/elastic/libbeat/logp"
)

type parser struct {
	parseOffset int
	// bytesReceived int
	message *redisMessage
}

type redisMessage struct {
	Ts time.Time

	TcpTuple     common.TcpTuple
	CmdlineTuple *common.CmdlineTuple
	Direction    uint8

	IsRequest bool
	IsError   bool
	Message   string
	Method    string
	Path      string
	Size      int

	next *redisMessage
}

const (
	START = iota
	BULK_ARRAY
	SIMPLE_MESSAGE
)

// Keep sorted for future command addition
var redisCommands = map[string]struct{}{
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

func isRedisCommand(key string) bool {
	_, exists := redisCommands[strings.ToUpper(key)]
	return exists
}

func (p *parser) reset() {
	p.parseOffset = 0
	p.message = nil
}

func (parser *parser) parse(buf *streambuf.Buffer) (bool, bool) {
	snapshot := buf.Snapshot()

	content, iserror, ok, complete := parser.dispatch(0, buf)
	if !ok || !complete {
		// on error or incomplete message drop all parsing progress, due to
		// parse not being statefull among multiple calls
		// => parser needs to restart parsing all content
		buf.Restore(snapshot)
		return ok, complete
	}

	parser.message.IsError = iserror
	parser.message.Size = buf.BufferConsumed()
	parser.message.Message = content
	return true, true
}

func (p *parser) dispatch(depth int, buf *streambuf.Buffer) (string, bool, bool, bool) {
	if buf.Len() == 0 {
		return "", false, true, false
	}

	var value string
	var iserror, ok, complete bool
	snapshot := buf.Snapshot()

	switch buf.Bytes()[0] {
	case '*':
		value, iserror, ok, complete = p.parseArray(depth, buf)
	case '$':
		value, ok, complete = p.parseString(buf)
	case ':':
		value, ok, complete = p.parseInt(buf)
	case '+':
		value, ok, complete = p.parseSimpleString(buf)
	case '-':
		iserror = true
		value, ok, complete = p.parseSimpleString(buf)
	default:
		debug("Unexpected message starting with %s", buf.Bytes()[0])
		return "", false, false, false
	}

	if !ok || !complete {
		buf.Restore(snapshot)
	}
	return value, iserror, ok, complete
}

func (p *parser) parseInt(buf *streambuf.Buffer) (string, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		return "", true, false
	}

	number := string(line[1:])
	if _, err := strconv.ParseInt(number, 10, 64); err != nil {
		logp.Err("Failed to read integer reply: %s", err)
	}

	return number, true, true
}

func (p *parser) parseSimpleString(buf *streambuf.Buffer) (string, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		return "", true, false
	}

	return string(line[1:]), true, true
}

func (p *parser) parseString(buf *streambuf.Buffer) (string, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		return "", true, false
	}

	if len(line) == 3 && line[1] == '-' && line[2] == '1' {
		return "nil", true, true
	}

	length, err := strconv.ParseInt(string(line[1:]), 10, 64)
	if err != nil {
		logp.Err("Failed to read bulk message: %s", err)
		return "", false, false
	}

	content, err := buf.CollectWithSuffix(int(length), []byte("\r\n"))
	if err != nil {
		if err != streambuf.ErrNoMoreBytes {
			return "", false, false
		}
		return "", true, false
	}

	return string(content), true, true
}

func (p *parser) parseArray(depth int, buf *streambuf.Buffer) (string, bool, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		debug("End of line not found, waiting for more data")
		return "", false, false, false
	}
	debug("line %s: %d", line, buf.BufferConsumed())

	if len(line) == 3 && line[1] == '-' && line[2] == '1' {
		return "nil", false, true, true
	}

	if len(line) == 2 && line[1] == '0' {
		return "[]", false, true, true
	}

	count, err := strconv.ParseInt(string(line[1:]), 10, 64)
	if err != nil {
		logp.Err("Failed to read number of bulk messages: %s", err)
		return "", false, false, false
	}
	if count < 0 {
		return "nil", false, true, true
	}

	content := make([]string, 0, count)
	// read sub elements

	iserror := false
	for i := 0; i < int(count); i++ {
		var value string
		var ok, complete bool

		value, iserror, ok, complete := p.dispatch(depth+1, buf)
		if !ok || !complete {
			debug("Array incomplete")
			return "", iserror, ok, complete
		}

		content = append(content, value)
	}

	if depth == 0 && isRedisCommand(content[0]) { // we've got a request
		p.message.IsRequest = true
		p.message.Method = content[0]
		p.message.Path = content[1]
	}

	var value string
	if depth == 0 && p.message.IsRequest {
		value = strings.Join(content, " ")
	} else {
		value = "[" + strings.Join(content, ", ") + "]"
	}
	return value, iserror, true, true
}
