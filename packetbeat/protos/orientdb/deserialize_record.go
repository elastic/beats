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

package orientdb

import (
	"errors"
	"fmt"
)

// RecordId Represents OrientDB RecordId
type RecordId struct {
	cluster  int16
	position int64
}

// Record Represents OrientDB Record
type Record struct {
	rid     RecordId
	oClass  string
	version int32
	oData   map[string]interface{}
}

func (record *Record) jsonSerialize() map[string]interface{} {
	meta := make(map[string]interface{})
	meta["rid"] = record.rid
	meta["version"] = record.version
	meta["oClass"] = record.oClass
	meta["oData"] = record.oData
	return meta
}

func deserialize(input []byte) *Record {
	length := len(input)
	if input == nil || len(input) == 0 {
		return nil
	}

	record := Record{}
	props := make(map[string]interface{})

	key, isClassName, idx := eatFirstKey(input)
	if idx < length {
		if isClassName {
			record.oClass = string(key)
		} else {
			input = input[idx:]
			chunk, index := eatValue(input)
			props[string(key)] = string(chunk)
			idx = index
		}
		if idx < length {
			input = input[idx:]

			first := true
			for len(input) > 0 {
				if (!first) && fmt.Sprintf("%c", input[0]) == "," {
					input = input[1:]
				} else if !first {
					break
				}

				key, idx := eatKey(input)
				input = input[idx:]
				if len(input) > 0 {
					chunk, idx := eatValue(input)
					if idx < len(input) {
						input = input[idx:]
					}
					props[string(key)] = string(chunk)
				}
				first = false
			}
		}
	}
	record.oData = props
	return &record
}

func eatFirstKey(input []byte) ([]byte, bool, int) {
	length := len(input)
	collected := make([]byte, 0)
	isClassName := false

	if fmt.Sprintf("%c", input[0]) == "\"" {
		result, _ := eatString(input[1:])
		return []byte{result[0]}, false, -1
	}

	i := 0
	for ; i < length; i++ {
		c := fmt.Sprintf("%c", input[i])
		if c == "@" {
			isClassName = true
			break
		} else if c == ":" {
			break
		} else {
			collected = append(collected, input[i])
		}
	}
	return collected, isClassName, i + 1
}

func eatKey(input []byte) ([]byte, int) {
	length := len(input)
	collected := make([]byte, 0)

	if length >= 1 && fmt.Sprintf("%c", input[0]) == "\"" {
		result, val := eatString(input[1:])
		return result, val
	}

	i := 0
	for ; i < length; i++ {
		c := fmt.Sprintf("%c", input[i])
		if c == ":" {
			break
		} else {
			collected = append(collected, input[i])
		}
	}
	return collected, i + 1
}

func eatValue(input []byte) ([]byte, int) {
	i := 0
	for ; i < len(input); i++ {
		if " " == fmt.Sprintf("%c", input[i]) {
			break
		}
	}
	if i+1 < len(input) {
		input = input[i+1:]
	}
	c := fmt.Sprintf("%c", input[0])
	if c == "," {
		return []byte{0}, 0
	}
	return eatString(input)
}

func eatString(input []byte) ([]byte, int) {
	length := len(input)
	collected := make([]byte, 0)
	i := 0
	for ; i < length; i++ {
		c := fmt.Sprintf("%c", input[i])
		if c == "\\" {
			i++
			collected = append(collected, input[i])
			continue
		} else if c == "\"" {
			break
		} else {
			collected = append(collected, input[i])
		}
	}
	return collected, i + 1
}

func unpackInt(input []byte) (int, error) {
	if len(input) != 4 {
		return 0, errors.New("input length not sufficient for int")
	}

	return int((uint32(input[3]) << 0) |
		(uint32(input[2]) << 8) |
		(uint32(input[1]) << 16) |
		(uint32(input[0]) << 24)), nil
}

func unpackString(input []byte) (string, error) {
	return string(input), nil
}
