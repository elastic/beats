package redis

import "testing"

func TestRedisParser_ArrayRequest(t *testing.T) {

	var message string
	message = "*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$5\r\nHello\r\n"

	stream := &RedisStream{data: []byte(message), message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if !stream.message.IsRequest {
		t.Errorf("Failed to parse Redis request")
	}
	if stream.message.Message != "SET key1 Hello" {
		t.Errorf("Failed to parse Redis request: %s", stream.message.Message)
	}
	if stream.message.Size != 34 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestRedisParser_ArrayResponse(t *testing.T) {

	var message string
	message = "*4\r\n$3\r\nfoo\r\n$-1\r\n$3\r\nbar\r\n:23\r\n"

	stream := &RedisStream{data: []byte(message), message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if stream.message.Message != "[foo, nil, bar, 23]" {
		t.Errorf("Failed to parse Redis request: %s", stream.message.Message)
	}
	if stream.message.Size != 32 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestRedisParser_SimpleString(t *testing.T) {

	var message string
	message = "+OK\r\n"

	stream := &RedisStream{data: []byte(message), message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if stream.message.Message != "OK" {
		t.Errorf("Failed to parse Redis response: %s", stream.message.Message)
	}
	if stream.message.Size != 5 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestRedisParser_NilString(t *testing.T) {

	var message string
	message = "$-1\r\n"

	stream := &RedisStream{data: []byte(message), message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if stream.message.Message != "nil" {
		t.Errorf("Failed to parse Redis response: %s", stream.message.Message)
	}
	if stream.message.Size != 5 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestRedisParser_EmptyString(t *testing.T) {

	var message string
	message = "$0\r\n\r\n"

	stream := &RedisStream{data: []byte(message), message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if stream.message.Message != "" {
		t.Errorf("Failed to parse Redis response: %s", stream.message.Message)
	}
	if stream.message.Size != 6 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

func TestRedisParser_EmptyArray(t *testing.T) {

	var message string
	message = "*0\r\n"

	stream := &RedisStream{data: []byte(message), message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse Redis response")
	}
	if stream.message.Message != "[]" {
		t.Errorf("Failed to parse Redis response: %s", stream.message.Message)
	}
	if stream.message.Size != 4 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}

}
