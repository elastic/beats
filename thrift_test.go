package main

import (
	"encoding/hex"
	"testing"
)

func TestThrift_thriftReadString(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var off int
	var str string

	data, _ = hex.DecodeString("0000000470696e67")
	str, ok, complete, off = thriftReadString(data)
	if str != "ping" || !ok || !complete || off != 8 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("0000000470696e670000")
	str, ok, complete, off = thriftReadString(data)
	if str != "ping" || !ok || !complete || off != 8 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("0000000470696e")
	str, ok, complete, off = thriftReadString(data)
	if str != "" || !ok || complete || off != 0 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}

	data, _ = hex.DecodeString("000000")
	str, ok, complete, off = thriftReadString(data)
	if str != "" || !ok || complete || off != 0 {
		t.Error("Bad result: %s %s %s %s", str, ok, complete, off)
	}
}

func TestThrift_readMessageBegin(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var stream ThriftStream
	var m *ThriftMessage

	data, _ = hex.DecodeString("800100010000000470696e670000000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftTypeCall ||
		m.SeqId != 0 || m.Version != ThriftVersion1 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftTypeCall ||
		m.SeqId != 0 || m.Version != ThriftVersion1 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000470696e6700000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftTypeCall ||
		m.SeqId != 1 || m.Version != ThriftVersion1 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result: %s %s", ok, complete)
	}

	data, _ = hex.DecodeString("800100010000000570696e6700000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result: %s %s", ok, complete)
	}

	data, _ = hex.DecodeString("0000000470696e670100000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || !complete {
		t.Error("Bad result: %s %s", ok, complete)
	}
	if m.Method != "ping" || m.Type != ThriftTypeCall ||
		m.SeqId != 0 || m.Version != 0 {
		t.Error("Bad values: %s %s %s %s", m.Method, m.Type, m.SeqId, m.Version)
	}

	data, _ = hex.DecodeString("0000000570696e670100000000")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	m = stream.message
	ok, complete = m.readMessageBegin(&stream)
	if !ok || complete {
		t.Error("Bad result:", ok, complete)
	}

}

func TestThrift_thriftReadField(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	var data []byte
	var ok, complete bool
	var stream ThriftStream
	var field *ThriftField
	var _old int

	data, _ = hex.DecodeString("08000100000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeI32 || field.Value != "1" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0600010001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeI16 || field.Value != "1" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0a00010000000000000001")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeI64 || field.Value != "1" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0400013ff333333333333300")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeDouble || field.Value != "1.2" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("02000101")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeBool || field.Value != "true" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c") // incomplete string
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field != nil {
		t.Error("Bad result:", ok, complete, field)
	}

	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeString || field.Value != "hello" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, ThriftStringMaxSize = ThriftStringMaxSize, 3
	data, _ = hex.DecodeString("0b00010000000568656c6c6f")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeString || field.Value != "hel..." {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	ThriftStringMaxSize = _old

	data, _ = hex.DecodeString("0f00010600000003000100020003")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeList ||
			field.Value != "[1, 2, 3]" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, ThriftListMaxSize = ThriftListMaxSize, 1
	data, _ = hex.DecodeString("0f00010600000003000100020003")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeList ||
			field.Value != "[1, ...]" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	ThriftListMaxSize = _old

	data, _ = hex.DecodeString("0e0001060000000300010002000300")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeSet ||
			field.Value != "{1, 2, 3}" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}

	_old, ThriftListMaxSize = ThriftListMaxSize, 2
	data, _ = hex.DecodeString("0e0001060000000300010002000300")
	stream = ThriftStream{tcpStream: nil, data: data, message: new(ThriftMessage)}
	ok, complete, field = thriftReadField(&stream)
	if !ok || complete || field == nil {
		t.Error("Bad result:", ok, complete, field)
	} else {
		if field.Id != 1 || field.Type != ThriftTypeSet ||
			field.Value != "{1, 2, ...}" {
			t.Error("Bad values:", field.Id, field.Type, field.Value)
		}
	}
	ThriftListMaxSize = _old

}

func TestThrift_simpleRequest(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	data := []byte(
		"800100010000000470696e670000000000",
	)

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &ThriftStream{tcpStream: nil, data: message, message: new(ThriftMessage)}

	ok, complete := thriftMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}
	if !stream.message.IsRequest {
		t.Error("Failed to parse Thrift request")
	}
	if stream.message.Method != "ping" {
		t.Error("Failed to parse query")
	}

}
