package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestStream(content []byte) *stream {
	st := newStream(time.Now(), nil)
	st.Append(content)
	return st
}

func parse(content []byte) (*redisMessage, bool, bool) {
	st := newTestStream(content)
	ok, complete := st.parser.parse(&st.Buf)
	return st.parser.message, ok, complete
}

func TestRedisParser_NoArgsRequest(t *testing.T) {
	message := []byte("*1\r\n" +
		"$4\r\n" +
		"INFO\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, msg.IsRequest)
	assert.Equal(t, "INFO", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_ArrayRequest(t *testing.T) {
	message := []byte("*3\r\n" +
		"$3\r\n" +
		"SET\r\n" +
		"$4\r\n" +
		"key1\r\n" +
		"$5\r\n" +
		"Hello\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, msg.IsRequest)
	assert.Equal(t, "SET key1 Hello", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_ArrayResponse(t *testing.T) {
	message := []byte("*4\r\n" +
		"$3\r\n" +
		"foo\r\n" +
		"$-1\r\n" +
		"$3\r\n" +
		"bar\r\n" +
		":23\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "[foo, nil, bar, 23]", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_ArrayNested(t *testing.T) {
	message := []byte("*3\r\n" +
		"*-1\r\n" +
		"+foo\r\n" +
		"*2\r\n" +
		":1\r\n" +
		"+bar\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "[nil, foo, [1, bar]]", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_SimpleString(t *testing.T) {
	message := []byte("+OK\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "OK", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_NilString(t *testing.T) {
	message := []byte("$-1\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "nil", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_EmptyString(t *testing.T) {
	message := []byte("$0\r\n\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_LenString(t *testing.T) {
	message := []byte("$5\r\n" +
		"12345\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "12345", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_LenStringWithCRLF(t *testing.T) {
	message := []byte("$7\r\n" +
		"123\r\n45\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "123\r\n45", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_EmptyArray(t *testing.T) {
	message := []byte("*0\r\n")
	msg, ok, complete := parse(message)

	assert.True(t, ok)
	assert.True(t, complete)
	assert.False(t, msg.IsRequest)
	assert.Equal(t, "[]", msg.Message)
	assert.Equal(t, len(message), msg.Size)
}

func TestRedisParser_Array2Passes(t *testing.T) {
	part1 := []byte("*3\r\n" +
		"$3\r\n" +
		"SET\r\n" +
		"$4\r\n")
	part2 := []byte("key1\r\n" +
		"$5\r\n" +
		"Hello\r\n")

	st := newTestStream(part1)
	ok, complete := st.parser.parse(&st.Buf)
	assert.True(t, ok)
	assert.False(t, complete)

	st.Stream.Append(part2)
	ok, complete = st.parser.parse(&st.Buf)
	msg := st.parser.message

	assert.True(t, ok)
	assert.True(t, complete)
	assert.True(t, msg.IsRequest)
	assert.Equal(t, "SET key1 Hello", msg.Message)
	assert.Equal(t, len(part1)+len(part2), msg.Size)
}
