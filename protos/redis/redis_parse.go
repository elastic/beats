package redis

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

type redisMessage struct {
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

func redisMessageParser(s *stream) (bool, bool) {
	var err error
	var value string
	m := s.message

	iserror := false

	for s.parseOffset < len(s.data) {

		if s.data[s.parseOffset] == '*' {
			//Arrays

			m.parseState = BULK_ARRAY
			m.start = s.parseOffset
			debug("start %d", m.start)

			found, line, off := readLine(s.data, s.parseOffset)
			if !found {
				debug("End of line not found, waiting for more data")
				return true, false
			}
			debug("line %s: %d", line, off)

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
				m.Bulks = make([]string, 0, m.NumberOfBulks)
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
				debug("End of line not found, waiting for more data")
				s.parseOffset = starting_offset
				return true, false
			}
			debug("line %s: %d", line, off)

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

				// check all content in buffer (length + CRLF)
				if int64(len(s.data[s.parseOffset:])) < length+2 {
					debug("Message incomplete, waiting for more data")
					s.parseOffset = starting_offset
					return true, false
				}

				// check content ends with CRLF
				off = s.parseOffset + int(length)
				if s.data[off] != '\r' || s.data[off+1] != '\n' {
					logp.Err("Expected end of line not found")
					return false, false
				}

				// extract line
				line = string(s.data[s.parseOffset:off])
				off += 2

				debug("line %s: %d", line, s.parseOffset)
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
			debug("Unexpected message starting with %s", s.data[s.parseOffset:])
			return false, false
		}

		// add value
		if m.NumberOfBulks > 0 {
			m.NumberOfBulks = m.NumberOfBulks - 1
			m.Bulks = append(m.Bulks, value)

			if len(m.Bulks) == 1 {
				debug("Value: %s", value)
				// first word.
				// check if it's a command
				if isRedisCommand(value) {
					debug("is request")
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
