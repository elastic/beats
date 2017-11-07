package redis

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
)

type parser struct {
	parseOffset int
	// bytesReceived int
	message *redisMessage
}

type redisMessage struct {
	ts time.Time

	tcpTuple     common.TCPTuple
	cmdlineTuple *common.CmdlineTuple
	direction    uint8

	isRequest bool
	isError   bool
	size      int
	message   common.NetString
	method    common.NetString
	path      common.NetString

	next *redisMessage
}

const (
	start = iota
	bulkArray
	simpleMessage
)

var (
	empty    = common.NetString("")
	emptyArr = common.NetString("[]")
	nilStr   = common.NetString("nil")
)

// Keep sorted for future command addition
var redisCommands = map[string]struct{}{
	"APPEND":           {},
	"AUTH":             {},
	"BGREWRITEAOF":     {},
	"BGSAVE":           {},
	"BITCOUNT":         {},
	"BITOP":            {},
	"BITPOS":           {},
	"BLPOP":            {},
	"BRPOP":            {},
	"BRPOPLPUSH":       {},
	"CLIENT GETNAME":   {},
	"CLIENT KILL":      {},
	"CLIENT LIST":      {},
	"CLIENT PAUSE":     {},
	"CLIENT SETNAME":   {},
	"CONFIG GET":       {},
	"CONFIG RESETSTAT": {},
	"CONFIG REWRITE":   {},
	"CONFIG SET":       {},
	"DBSIZE":           {},
	"DEBUG OBJECT":     {},
	"DEBUG SEGFAULT":   {},
	"DECR":             {},
	"DECRBY":           {},
	"DEL":              {},
	"DISCARD":          {},
	"DUMP":             {},
	"ECHO":             {},
	"EVAL":             {},
	"EVALSHA":          {},
	"EXEC":             {},
	"EXISTS":           {},
	"EXPIRE":           {},
	"EXPIREAT":         {},
	"FLUSHALL":         {},
	"FLUSHDB":          {},
	"GET":              {},
	"GETBIT":           {},
	"GETRANGE":         {},
	"GETSET":           {},
	"HDEL":             {},
	"HEXISTS":          {},
	"HGET":             {},
	"HGETALL":          {},
	"HINCRBY":          {},
	"HINCRBYFLOAT":     {},
	"HKEYS":            {},
	"HLEN":             {},
	"HMGET":            {},
	"HMSET":            {},
	"HSCAN":            {},
	"HSET":             {},
	"HSETINX":          {},
	"HVALS":            {},
	"INCR":             {},
	"INCRBY":           {},
	"INCRBYFLOAT":      {},
	"INFO":             {},
	"KEYS":             {},
	"LASTSAVE":         {},
	"LINDEX":           {},
	"LINSERT":          {},
	"LLEN":             {},
	"LPOP":             {},
	"LPUSH":            {},
	"LPUSHX":           {},
	"LRANGE":           {},
	"LREM":             {},
	"LSET":             {},
	"LTRIM":            {},
	"MGET":             {},
	"MIGRATE":          {},
	"MONITOR":          {},
	"MOVE":             {},
	"MSET":             {},
	"MSETNX":           {},
	"MULTI":            {},
	"OBJECT":           {},
	"PERSIST":          {},
	"PEXPIRE":          {},
	"PEXPIREAT":        {},
	"PFADD":            {},
	"PFCOUNT":          {},
	"PFMERGE":          {},
	"PING":             {},
	"PSETEX":           {},
	"PSUBSCRIBE":       {},
	"PTTL":             {},
	"PUBLISH":          {},
	"PUBSUB":           {},
	"PUNSUBSCRIBE":     {},
	"QUIT":             {},
	"RANDOMKEY":        {},
	"RENAME":           {},
	"RENAMENX":         {},
	"RESTORE":          {},
	"RPOP":             {},
	"RPOPLPUSH":        {},
	"RPUSH":            {},
	"RPUSHX":           {},
	"SADD":             {},
	"SAVE":             {},
	"SCAN":             {},
	"SCARD":            {},
	"SCRIPT EXISTS":    {},
	"SCRIPT FLUSH":     {},
	"SCRIPT KILL":      {},
	"SCRIPT LOAD":      {},
	"SDIFF":            {},
	"SDIFFSTORE":       {},
	"SELECT":           {},
	"SET":              {},
	"SETBIT":           {},
	"SETEX":            {},
	"SETNX":            {},
	"SETRANGE":         {},
	"SHUTDOWN":         {},
	"SINTER":           {},
	"SINTERSTORE":      {},
	"SISMEMBER":        {},
	"SLAVEOF":          {},
	"SLOWLOG":          {},
	"SMEMBERS":         {},
	"SMOVE":            {},
	"SORT":             {},
	"SPOP":             {},
	"SRANDMEMBER":      {},
	"SREM":             {},
	"SSCAN":            {},
	"STRLEN":           {},
	"SUBSCRIBE":        {},
	"SUNION":           {},
	"SUNIONSTORE":      {},
	"SYNC":             {},
	"TIME":             {},
	"TTL":              {},
	"TYPE":             {},
	"UNSUBSCRIBE":      {},
	"UNWATCH":          {},
	"WATCH":            {},
	"ZADD":             {},
	"ZCARD":            {},
	"ZCOUNT":           {},
	"ZINCRBY":          {},
	"ZINTERSTORE":      {},
	"ZRANGE":           {},
	"ZRANGEBYSCORE":    {},
	"ZRANK":            {},
	"ZREM":             {},
	"ZREMRANGEBYLEX":   {},
	"ZREMRANGEBYRANK":  {},
	"ZREMRANGEBYSCORE": {},
	"ZREVRANGE":        {},
	"ZREVRANGEBYSCORE": {},
	"ZREVRANK":         {},
	"ZSCAN":            {},
	"ZSCORE":           {},
	"ZUNIONSTORE":      {},
}

var maxCommandLen = 0

const commandLenBuffer = 50

func init() {
	for k := range redisCommands {
		l := len(k)
		if l > maxCommandLen {
			maxCommandLen = l
		}
	}

	// panic normally triggered during testing to give a note about small buffer sizes
	if maxCommandLen > commandLenBuffer {
		panic("commandLenBuffer small")
	}
}

func isRedisCommand(key common.NetString) bool {
	if len(key) > maxCommandLen {
		return false
	}

	// key to upper into pre-allocated buffer (commands use ASCII only)
	var buf [commandLenBuffer]byte
	upper := buf[:len(key)]
	for i, b := range key {
		if 'a' <= b && b <= 'z' {
			b = b - 'a' + 'A'
		}
		upper[i] = b
	}

	_, exists := redisCommands[string(upper)]
	return exists
}

func (p *parser) reset() {
	p.parseOffset = 0
	p.message = nil
}

func (p *parser) parse(buf *streambuf.Buffer) (bool, bool) {
	snapshot := buf.Snapshot()

	content, iserror, ok, complete := p.dispatch(0, buf)
	if !ok || !complete {
		// on error or incomplete message drop all parsing progress, due to
		// parse not being statefull among multiple calls
		// => parser needs to restart parsing all content
		buf.Restore(snapshot)
		return ok, complete
	}

	p.message.isError = iserror
	p.message.size = buf.BufferConsumed()
	p.message.message = content
	return true, true
}

func (p *parser) dispatch(depth int, buf *streambuf.Buffer) (common.NetString, bool, bool, bool) {
	if buf.Len() == 0 {
		return empty, false, true, false
	}

	var value common.NetString
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
		if isDebug {
			debugf("Unexpected message starting with %s", buf.Bytes()[0])
		}
		return empty, false, false, false
	}

	if !ok || !complete {
		buf.Restore(snapshot)
	}
	return value, iserror, ok, complete
}

func (p *parser) parseInt(buf *streambuf.Buffer) (common.NetString, bool, bool) {
	value, ok, complete := p.parseSimpleString(buf)
	if ok && complete {
		if _, err := parseInt(value); err != nil {
			logp.Err("Failed to read integer reply: %s", err)
		}
	}
	return value, ok, complete
}

func (p *parser) parseSimpleString(buf *streambuf.Buffer) (common.NetString, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		return empty, true, false
	}

	return common.NetString(line[1:]), true, true
}

func (p *parser) parseString(buf *streambuf.Buffer) (common.NetString, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		return empty, true, false
	}

	if len(line) == 3 && line[1] == '-' && line[2] == '1' {
		return nilStr, true, true
	}

	length, err := parseInt(line[1:])
	if err != nil {
		logp.Err("Failed to read bulk message: %s", err)
		return empty, false, false
	}

	content, err := buf.CollectWithSuffix(int(length), []byte("\r\n"))
	if err != nil {
		if err != streambuf.ErrNoMoreBytes {
			return common.NetString{}, false, false
		}
		return common.NetString{}, true, false
	}

	return common.NetString(content), true, true
}

func (p *parser) parseArray(depth int, buf *streambuf.Buffer) (common.NetString, bool, bool, bool) {
	line, err := buf.UntilCRLF()
	if err != nil {
		if isDebug {
			debugf("End of line not found, waiting for more data")
		}
		return empty, false, false, false
	}
	if isDebug {
		debugf("line %s: %d", line, buf.BufferConsumed())
	}

	if len(line) == 3 && line[1] == '-' && line[2] == '1' {
		return nilStr, false, true, true
	}

	if len(line) == 2 && line[1] == '0' {
		return emptyArr, false, true, true
	}

	count, err := parseInt(line[1:])
	if err != nil {
		logp.Err("Failed to read number of bulk messages: %s", err)
		return empty, false, false, false
	}
	if count < 0 {
		return nilStr, false, true, true
	} else if count == 0 {
		// should not happen, but handle just in case ParseInt did return 0
		return emptyArr, false, true, true
	}

	// invariant: count > 0

	// try to allocate content array right on stack
	var content [][]byte
	const arrayBufferSize = 32
	if int(count) <= arrayBufferSize {
		var arrayBuffer [arrayBufferSize][]byte
		content = arrayBuffer[:0]
	} else {
		content = make([][]byte, 0, count)
	}

	contentLen := 0
	// read sub elements

	iserror := false
	for i := 0; i < int(count); i++ {
		var value common.NetString
		var ok, complete bool

		value, iserror, ok, complete := p.dispatch(depth+1, buf)
		if !ok || !complete {
			if isDebug {
				debugf("Array incomplete")
			}
			return empty, iserror, ok, complete
		}

		content = append(content, []byte(value))
		contentLen += len(value)
	}

	// handle top-level request command
	if depth == 0 && isRedisCommand(content[0]) {
		p.message.isRequest = true
		p.message.method = content[0]
		if len(content) > 1 {
			p.message.path = content[1]
		}

		var value common.NetString
		if contentLen > 1 {
			tmp := make([]byte, contentLen+(len(content)-1)*1)
			join(tmp, content, []byte(" "))
			value = common.NetString(tmp)
		} else {
			value = common.NetString(content[0])
		}
		return value, iserror, true, true
	}

	// return redis array: [a, b, c]
	tmp := make([]byte, 2+contentLen+(len(content)-1)*2)
	tmp[0] = '['
	join(tmp[1:], content, []byte(", "))
	tmp[len(tmp)-1] = ']'
	value := common.NetString(tmp)
	return value, iserror, true, true
}

func parseInt(line []byte) (int64, error) {
	buf := streambuf.NewFixed(line)
	return buf.IntASCII(false)
	// TODO: is it an error if 'buf.Len() != 0 {}' ?
}

func join(to []byte, content [][]byte, sep []byte) {
	if len(content) > 0 {
		off := copy(to, content[0])
		for i := 1; i < len(content); i++ {
			off += copy(to[off:], sep)
			off += copy(to[off:], content[i])
		}
	}
}
