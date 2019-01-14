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

// +build !integration

package orientdb

import "testing"

func TestOrientdbParser_messageNotEvenStarted(t *testing.T) {
	var data []byte

	st := &stream{data: data, message: new(orientdbMessage)}

	ok, complete := orientdbMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if complete {
		t.Errorf("Expecting an incomplete message")
	}
}

func TestOrientdbParser_simpleRequest(t *testing.T) {
	var data []byte
	data = addByte(data, 74) // opCode = 74 = REQUEST_DB_LIST
	data = addInt32(data, 1) // sessionID = 1

	st := &stream{data: data, message: new(orientdbMessage)}

	ok, complete := orientdbMessageParser(st)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
}

func TestOrientdbParser_unknownOpCode(t *testing.T) {
	var data []byte
	data = addByte(data, 0)

	st := &stream{data: data, message: new(orientdbMessage)}

	ok, complete := orientdbMessageParser(st)

	if ok {
		t.Errorf("Parsing should have returned an error")
	}

	if complete {
		t.Errorf("Not expecting a complete message")
	}
}

func addByte(in []byte, v byte) []byte {
	return append(in, v)
}

func addInt32(in []byte, v int32) []byte {
	u := uint32(v)
	return append(in, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
}
