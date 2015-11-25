package redis

import "testing"

func TestRedisParser_ArrayRequest(t *testing.T) {

	message := []byte("*3\r\n" +
		"$3\r\n" +
		"SET\r\n" +
		"$4\r\n" +
		"key1\r\n" +
		"$5\r\n" +
		"Hello\r\n")

	st := &stream{data: message, message: new(redisMessage)}

	ok, complete := redisMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if !st.message.IsRequest {
		t.Errorf("Failed to parse Redis request")
	}
	if st.message.Message != "SET key1 Hello" {
		t.Errorf("Failed to parse Redis request: %s", st.message.Message)
	}
	if st.message.Size != 34 {
		t.Errorf("Wrong message size %d", st.message.Size)
	}
}

func TestRedisParser_ArrayResponse(t *testing.T) {

	message := []byte("*4\r\n" +
		"$3\r\n" +
		"foo\r\n" +
		"$-1\r\n" +
		"$3\r\n" +
		"bar\r\n" +
		":23\r\n")

	st := &stream{data: message, message: new(redisMessage)}

	ok, complete := redisMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if st.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if st.message.Message != "[foo, nil, bar, 23]" {
		t.Errorf("Failed to parse Redis request: %s", st.message.Message)
	}
	if st.message.Size != 32 {
		t.Errorf("Wrong message size %d", st.message.Size)
	}
}

func TestRedisParser_SimpleString(t *testing.T) {

	message := []byte("+OK\r\n")

	st := &stream{data: message, message: new(redisMessage)}

	ok, complete := redisMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if st.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if st.message.Message != "OK" {
		t.Errorf("Failed to parse Redis response: %s", st.message.Message)
	}
	if st.message.Size != 5 {
		t.Errorf("Wrong message size %d", st.message.Size)
	}
}

func TestRedisParser_NilString(t *testing.T) {

	message := []byte("$-1\r\n")

	st := &stream{data: message, message: new(redisMessage)}

	ok, complete := redisMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if st.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if st.message.Message != "nil" {
		t.Errorf("Failed to parse Redis response: %s", st.message.Message)
	}
	if st.message.Size != 5 {
		t.Errorf("Wrong message size %d", st.message.Size)
	}
}

func TestRedisParser_EmptyString(t *testing.T) {

	message := []byte("$0\r\n\r\n")

	st := &stream{data: message, message: new(redisMessage)}

	ok, complete := redisMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if st.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if st.message.Message != "" {
		t.Errorf("Failed to parse Redis response: %s", st.message.Message)
	}
	if st.message.Size != 6 {
		t.Errorf("Wrong message size %d", st.message.Size)
	}
}

func TestRedisParser_EmptyArray(t *testing.T) {

	message := []byte("*0\r\n")

	st := &stream{data: message, message: new(redisMessage)}

	ok, complete := redisMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if st.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if st.message.Message != "[]" {
		t.Errorf("Failed to parse Redis response: %s", st.message.Message)
	}
	if st.message.Size != 4 {
		t.Errorf("Wrong message size %d", st.message.Size)
	}
}
