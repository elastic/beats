package mongodb

import "testing"

func TestMongodbParser_messageNotEvenStarted(t *testing.T) {
	var data []byte
	data = append(data, 0)

	stream := &MongodbStream{data: data, message: new(MongodbMessage)}

	ok, complete := mongodbMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if complete {
		t.Errorf("Expecting an incomplete message")
	}
}

func TestMongodbParser_mesageNotFinished(t *testing.T) {
	var data []byte
	addInt32(data, 100) // length = 100

	stream := &MongodbStream{data: data, message: new(MongodbMessage)}

	ok, complete := mongodbMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if complete {
		t.Errorf("Expecting an incomplete message")
	}
}

func TestMongodbParser_simpleRequest(t *testing.T) {
	var data []byte
	data = addInt32(data, 26)   // length = 16 (header) + 9 (message) + 1 (message length)
	data = addInt32(data, 1)    // requestId = 1
	data = addInt32(data, 0)    // responseTo = 0
	data = addInt32(data, 1000) // opCode = 1000 = OP_MSG
	data = addCStr(data, "a message")

	stream := &MongodbStream{data: data, message: new(MongodbMessage)}

	ok, complete := mongodbMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestMongodbParser_unknownOpCode(t *testing.T) {
	var data []byte
	data = addInt32(data, 16)   // length = 16
	data = addInt32(data, 1)    // requestId = 1
	data = addInt32(data, 0)    // responseTo = 0
	data = addInt32(data, 5555) // opCode = 5555 = not a valid code

	stream := &MongodbStream{data: data, message: new(MongodbMessage)}

	ok, complete := mongodbMessageParser(stream)

	if ok {
		t.Errorf("Parsing should have returned an error")
	}
	if complete {
		t.Errorf("Not expecting a complete message")
	}
}

func addCStr(in []byte, v string) []byte {
	out := append(in, []byte(v)...)
	out = append(out, 0)
	return out
}

func addInt32(in []byte, v int32) []byte {
	u := uint32(v)
	return append(in, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}
