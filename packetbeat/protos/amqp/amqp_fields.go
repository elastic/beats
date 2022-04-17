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

package amqp

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// getTable updates fields with the table data at the given offset.
// fields must be non_nil on entry.
func getTable(fields common.MapStr, data []byte, offset uint32) (next uint32, err bool, exists bool) {
	length := binary.BigEndian.Uint32(data[offset : offset+4])

	// size declared too big
	if length > uint32(len(data[offset+4:])) {
		return 0, true, false
	}
	if length > 0 {
		exists = true
		table := common.MapStr{}
		err := fieldUnmarshal(table, data[offset+4:offset+4+length], 0, length, -1)
		if err {
			logp.Warn("Error while parsing a field table")
			return 0, true, false
		}
		fields.Update(table)
	}
	return length + 4 + offset, false, exists
}

// getTable updates fields with the array data at the given offset.
// fields must be non_nil on entry.
func getArray(fields common.MapStr, data []byte, offset uint32) (next uint32, err bool, exists bool) {
	length := binary.BigEndian.Uint32(data[offset : offset+4])

	// size declared too big
	if length > uint32(len(data[offset+4:])) {
		return 0, true, false
	}
	if length > 0 {
		exists = true
		array := common.MapStr{}
		err := fieldUnmarshal(array, data[offset+4:offset+4+length], 0, length, 0)
		if err {
			logp.Warn("Error while parsing a field array")
			return 0, true, false
		}
		fields.Update(array)
	}
	return length + 4 + offset, false, exists
}

// The index parameter, when set at -1, indicates that the entry is a field table.
// If it's set at 0, it is an array.
func fieldUnmarshal(table common.MapStr, data []byte, offset uint32, length uint32, index int) (err bool) {
	var name string

	if offset >= length {
		return false
	}
	// get name of the field. If it's an array, it will be the index parameter as a
	// string. If it's a table, it will be the name of the field.
	if index < 0 {
		fieldName, offsetTemp, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get short string in table")
			return true
		}
		name = fieldName
		offset = offsetTemp
	} else {
		name = strconv.Itoa(index)
		index++
	}

	switch data[offset] {
	case boolean:
		if data[offset+1] == 1 {
			table[name] = true
		} else {
			table[name] = false
		}
		offset += 2
	case shortShortInt:
		table[name] = int8(data[offset+1])
		offset += 2
	case shortShortUint:
		table[name] = uint8(data[offset+1])
		offset += 2
	case shortInt:
		table[name] = int16(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
		offset += 3
	case shortUint:
		table[name] = binary.BigEndian.Uint16(data[offset+1 : offset+3])
		offset += 3
	case longInt:
		table[name] = int(binary.BigEndian.Uint32(data[offset+1 : offset+5]))
		offset += 5
	case longUint:
		table[name] = binary.BigEndian.Uint32(data[offset+1 : offset+5])
		offset += 5
	case longLongInt:
		table[name] = int64(binary.BigEndian.Uint64(data[offset+1 : offset+9]))
		offset += 9
	case longLongUint:
		table[name] = binary.BigEndian.Uint64(data[offset+1 : offset+9])
		offset += 9
	case float:
		bits := binary.BigEndian.Uint32(data[offset+1 : offset+5])
		table[name] = math.Float32frombits(bits)
		offset += 5
	case double:
		bits := binary.BigEndian.Uint64(data[offset+1 : offset+9])
		table[name] = math.Float64frombits(bits)
		offset += 9
	case decimal:
		scale := data[offset+1]
		val := strings.Split(strconv.Itoa(int(binary.BigEndian.Uint32(data[offset+2:offset+6]))), "")
		ret := make([]string, len(val)+1)
		for i, j := 0, 0; i < len(val); i++ {
			if i == len(val)-int(scale) {
				ret[j] = "."
				j++
			}
			ret[j] = val[i]
			j++
		}
		table[name] = strings.Join(ret, "")
		offset += 6
	case shortString:
		s, next, err := getShortString(data, offset+2, uint32(data[offset+1]))
		if err {
			logp.Warn("Failed to get short string in table")
			return true
		}
		table[name] = s
		offset = next
	case longString:
		s, next, err := getShortString(data, offset+5, binary.BigEndian.Uint32(data[offset+1:offset+5]))
		if err {
			logp.Warn("Failed to get long string in table")
			return true
		}
		table[name] = s
		offset = next
	case fieldArray:
		newMap := common.MapStr{}
		next, err, _ := getArray(newMap, data, offset+1)
		if err {
			return true
		}
		table[name] = newMap
		offset = next
	case timestamp:
		t := time.Unix(int64(binary.BigEndian.Uint64(data[offset+1:offset+9])), 0)
		table[name] = t.Format(amqpTimeLayout)
		offset += 9
	case fieldTable:
		newMap := common.MapStr{}
		next, err, _ := getTable(newMap, data, offset+1)
		if err {
			return true
		}
		table[name] = newMap
		offset = next
	case noField:
		table[name] = nil
		offset++
	case byteArray:
		size := binary.BigEndian.Uint32(data[offset+1 : offset+5])
		table[name] = bodyToByteArray(data[offset+1+size : offset+5+size])
		offset += 5 + size
	default:
		// unknown field
		return true
	}
	// advance to next field recursively
	return fieldUnmarshal(table, data, offset, length, index)
}

// function to convert a body slice into a byte array
func bodyToByteArray(data []byte) string {
	ret := make([]string, len(data))
	for i, c := range data {
		ret[i] = strconv.Itoa(int(c))
	}
	return "[" + strings.Join(ret, ", ") + "]"
}
