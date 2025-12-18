// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration

package mongodb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMongodbParser_messageNotEvenStarted(t *testing.T) {
	var data []byte
	data = append(data, 0)

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if complete {
		t.Errorf("Expecting an incomplete message")
	}
}

func TestMongodbParser_messageNotFinished(t *testing.T) {
	var data []byte
	addInt32(data, 100) // length = 100

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

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

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestMongodbParser_OpMsg(t *testing.T) {
	files := []string{
		"1req.bin",
		"1res.bin",
		"2req.bin",
		"2req.bin",
		"3req.bin",
		"3res.bin",
	}

	for _, fn := range files {
		data, err := os.ReadFile(filepath.Join("testdata", fn))
		if err != nil {
			t.Fatal(err)
		}

		st := &stream{data: data, message: new(mongodbMessage)}

		ok, complete := mongodbMessageParser(st)

		if !ok {
			t.Errorf("Parsing returned error")
		}
		if !complete {
			t.Errorf("Expecting a complete message")
		}
		_, err = json.Marshal(st.message.documents)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMongodbParser_unknownOpCode(t *testing.T) {
	var data []byte
	data = addInt32(data, 16)   // length = 16
	data = addInt32(data, 1)    // requestId = 1
	data = addInt32(data, 0)    // responseTo = 0
	data = addInt32(data, 5555) // opCode = 5555 = not a valid code

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

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

func Test_isDatabaseCommand(t *testing.T) {
	type io struct {
		Key   string
		Value interface{}

		Output bool
	}
	tests := []io{
		{
			Key:    "listCollections",
			Value:  float64(1),
			Output: true,
		},
		{
			Key:    "listcollections",
			Value:  float64(1),
			Output: true,
		},
		{
			Key:    "findandmodify",
			Value:  "restaurants",
			Output: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, isDatabaseCommand(test.Key, test.Value))
	}
}

// Helper to add int64 in little-endian format
func addInt64(in []byte, v int64) []byte {
	u := uint64(v)
	return append(in, byte(u), byte(u>>8), byte(u>>16), byte(u>>24),
		byte(u>>32), byte(u>>40), byte(u>>48), byte(u>>56))
}

// Security test: Negative message length should be rejected without panic
func TestMongodbParser_negativeMessageLength(t *testing.T) {
	// messageLength = 0x80000000 (-2147483648 as int32) in little-endian
	// This would cause a panic in truncate() if not validated
	data := []byte{0x00, 0x00, 0x00, 0x80}

	st := &stream{data: data, message: new(mongodbMessage)}

	// Should not panic - use a deferred recover to catch any panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parser panicked on negative message length: %v", r)
		}
	}()

	ok, complete := mongodbMessageParser(st)

	// Should reject the message (ok=false) without completing
	assert.False(t, ok, "Parser should reject negative message length")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: Message length less than header size should be rejected
func TestMongodbParser_messageLengthTooSmall(t *testing.T) {
	// Length = 10, which is less than minimum header size (16 bytes)
	var data []byte
	data = addInt32(data, 10) // length too small
	data = addInt32(data, 1)  // requestId
	data = addInt32(data, 0)  // responseTo
	data = addInt32(data, 1)  // opCode (OP_REPLY)

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

	assert.False(t, ok, "Parser should reject message length smaller than header")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: Negative BSON document length should be rejected without panic
func TestMongodbParser_negativeBSONLength(t *testing.T) {
	// Build minimal OP_QUERY with a BSON doc length = -1 (0xFFFFFFFF)
	var data []byte
	data = addInt32(data, 0)    // placeholder for total length
	data = addInt32(data, 1)    // requestId
	data = addInt32(data, 0)    // responseTo
	data = addInt32(data, 2004) // opCode = OP_QUERY
	data = addInt32(data, 0)    // flags
	data = addCStr(data, "db.$cmd")
	data = addInt32(data, 0)                          // numberToSkip
	data = addInt32(data, 1)                          // numberToReturn
	data = append(data, 0xFF, 0xFF, 0xFF, 0xFF)       // BSON length = -1
	data = append(data, 0x00, 0x00, 0x00, 0x00, 0x00) // minimal invalid doc body

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parser panicked on negative BSON length: %v", r)
		}
	}()

	ok, complete := mongodbMessageParser(st)

	assert.False(t, ok, "Parser should reject negative BSON document length")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: BSON document length of zero should be rejected
func TestMongodbParser_zeroBSONLength(t *testing.T) {
	var data []byte
	data = addInt32(data, 0)    // placeholder for total length
	data = addInt32(data, 1)    // requestId
	data = addInt32(data, 0)    // responseTo
	data = addInt32(data, 2004) // opCode = OP_QUERY
	data = addInt32(data, 0)    // flags
	data = addCStr(data, "db.$cmd")
	data = addInt32(data, 0) // numberToSkip
	data = addInt32(data, 1) // numberToReturn
	data = addInt32(data, 0) // BSON length = 0 (invalid, minimum is 5)

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

	assert.False(t, ok, "Parser should reject zero BSON document length")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: Negative numberReturned in OP_REPLY should be rejected without panic
func TestMongodbParser_negativeNumberReturned(t *testing.T) {
	var data []byte
	data = addInt32(data, 0)  // placeholder for total length
	data = addInt32(data, 1)  // requestId
	data = addInt32(data, 1)  // responseTo (non-zero to indicate response)
	data = addInt32(data, 1)  // opCode = OP_REPLY
	data = addInt32(data, 0)  // flags
	data = addInt64(data, 0)  // cursorId
	data = addInt32(data, 0)  // startingFrom
	data = addInt32(data, -1) // numberReturned = -1 (would panic on make())

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parser panicked on negative numberReturned: %v", r)
		}
	}()

	ok, complete := mongodbMessageParser(st)

	assert.False(t, ok, "Parser should reject negative numberReturned")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: Excessively large numberReturned should be rejected without OOM
func TestMongodbParser_hugeNumberReturned(t *testing.T) {
	var data []byte
	data = addInt32(data, 0)          // placeholder for total length
	data = addInt32(data, 1)          // requestId
	data = addInt32(data, 1)          // responseTo
	data = addInt32(data, 1)          // opCode = OP_REPLY
	data = addInt32(data, 0)          // flags
	data = addInt64(data, 0)          // cursorId
	data = addInt32(data, 0)          // startingFrom
	data = addInt32(data, 0x3FFFFFFF) // numberReturned = ~1 billion (would cause OOM)
	// Add one minimal empty BSON document (5 bytes)
	data = append(data, 5, 0, 0, 0, 0)

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parser panicked on huge numberReturned: %v", r)
		}
	}()

	ok, complete := mongodbMessageParser(st)

	// Should reject because numberReturned exceeds what could fit in remaining bytes
	assert.False(t, ok, "Parser should reject numberReturned exceeding buffer capacity")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: Negative document sequence size in OP_MSG should be rejected
func TestMongodbParser_negativeOpMsgSequenceSize(t *testing.T) {
	var data []byte
	data = addInt32(data, 0)    // placeholder for total length
	data = addInt32(data, 1)    // requestId
	data = addInt32(data, 0)    // responseTo
	data = addInt32(data, 2013) // opCode = OP_MSG
	data = addInt32(data, 0)    // flagBits
	data = append(data, 1)      // kind = 1 (document sequence)
	data = addInt32(data, -1)   // size = -1 (negative)
	data = addCStr(data, "docs")

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parser panicked on negative OP_MSG sequence size: %v", r)
		}
	}()

	ok, complete := mongodbMessageParser(st)

	assert.False(t, ok, "Parser should reject negative OP_MSG document sequence size")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: OP_MSG sequence size exceeding buffer should be rejected
func TestMongodbParser_opMsgSequenceSizeExceedsBuffer(t *testing.T) {
	var data []byte
	data = addInt32(data, 0)       // placeholder for total length
	data = addInt32(data, 1)       // requestId
	data = addInt32(data, 0)       // responseTo
	data = addInt32(data, 2013)    // opCode = OP_MSG
	data = addInt32(data, 0)       // flagBits
	data = append(data, 1)         // kind = 1 (document sequence)
	data = addInt32(data, 1000000) // size = 1MB (way more than buffer has)
	data = addCStr(data, "docs")

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

	assert.False(t, ok, "Parser should reject OP_MSG sequence size exceeding buffer")
	assert.False(t, complete, "Message should not be complete")
}

// Security test: Valid OP_REPLY with correct numberReturned should still work
func TestMongodbParser_validOpReply(t *testing.T) {
	var data []byte
	data = addInt32(data, 0) // placeholder for total length
	data = addInt32(data, 1) // requestId
	data = addInt32(data, 1) // responseTo
	data = addInt32(data, 1) // opCode = OP_REPLY
	data = addInt32(data, 0) // flags
	data = addInt64(data, 0) // cursorId
	data = addInt32(data, 0) // startingFrom
	data = addInt32(data, 1) // numberReturned = 1
	// Add one minimal empty BSON document (5 bytes: length + null terminator)
	data = append(data, 5, 0, 0, 0, 0)

	// Fix total length
	tl := int32(len(data))
	data[0], data[1], data[2], data[3] = byte(tl), byte(tl>>8), byte(tl>>16), byte(tl>>24)

	st := &stream{data: data, message: new(mongodbMessage)}

	ok, complete := mongodbMessageParser(st)

	assert.True(t, ok, "Parser should accept valid OP_REPLY")
	assert.True(t, complete, "Message should be complete")
	assert.Equal(t, int32(1), st.message.event["numberReturned"])
}
