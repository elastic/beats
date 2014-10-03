package main

import (
	"encoding/hex"
	"testing"
	//"fmt"
	//"log/syslog"
)

func TestRedisParser_simpleRequest(t *testing.T) {

	data := []byte(
		"2a330d0a24330d0a5345540d0a24340d0a6b6579310d0a24350d0a48656c6c6f0d0a")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &RedisStream{tcpStream: nil, data: message, message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if !stream.message.IsRequest {
		t.Error("Failed to parse Redis request")
	}
	if stream.message.Message != "SET key1 Hello" {
		t.Error("Failed to parse Redis request: %s", stream.message.Message)
	}
}

func TestRedisParser_PosResult(t *testing.T) {

	data := []byte(
		"2b4f4b0d0a")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &RedisStream{tcpStream: nil, data: message, message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Error("Failed to parse Redis response")
	}
	if stream.message.Message != "OK" {
		t.Error("Failed to parse Redis response: %s", stream.message.Message)
	}
}

func TestRedisParser_NilResult(t *testing.T) {

	data := []byte(
		"2a310d0a242d310d0a")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &RedisStream{tcpStream: nil, data: message, message: new(RedisMessage)}

	ok, complete := redisMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Error("Failed to parse Redis response")
	}
	if stream.message.Message != "nil" {
		t.Error("Failed to parse Redis response: %s", stream.message.Message)
	}
}
