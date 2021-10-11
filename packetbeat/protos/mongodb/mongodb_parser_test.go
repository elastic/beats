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
// +build !integration

package mongodb

import (
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

func Test_extract_documents(t *testing.T) {
	type io struct {
		Input  map[string]interface{}
		Output []interface{}
	}
	tests := []io{
		{
			Input: map[string]interface{}{
				"a":         1,
				"documents": []interface{}{"a", "b", "c"},
			},
			Output: []interface{}{"a", "b", "c"},
		},
		{
			Input: map[string]interface{}{
				"a": 1,
			},
			Output: []interface{}{},
		},
		{
			Input: map[string]interface{}{
				"a":         1,
				"documents": 1,
			},
			Output: []interface{}{},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output, extractDocuments(test.Input))
	}
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
