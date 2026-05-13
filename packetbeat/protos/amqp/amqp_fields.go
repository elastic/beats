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

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// getTable updates fields with the table data at the given offset.
// fields must be non_nil on entry.
func getTable(fields mapstr.M, data []byte, offset uint32) (next uint32, err bool, exists bool) {
	offset64 := int64(offset)
	dataLen64 := int64(len(data))
	if offset64 > dataLen64 || 4 > dataLen64-offset64 {
		logp.Debug("amqp", "Error while parsing a field table")
		return 0, true, false
	}
	offsetInt := int(offset64)
	length := binary.BigEndian.Uint32(data[offsetInt : offsetInt+4])
	tableStart64 := offset64 + 4

	// size declared too big
	if int64(length) > dataLen64-tableStart64 {
		return 0, true, false
	}
	if length > ^uint32(0)-offset-4 {
		return 0, true, false
	}
	if length > 0 {
		exists = true
		table := mapstr.M{}
		tableStartInt := int(tableStart64)
		tableEndInt := tableStartInt + int(length)
		err := fieldUnmarshal(table, data[tableStartInt:tableEndInt], 0, length, -1)
		if err {
			logp.Debug("amqp", "Error while parsing a field table")
			return 0, true, false
		}
		fields.Update(table)
	}
	return offset + 4 + length, false, exists
}

// getTable updates fields with the array data at the given offset.
// fields must be non_nil on entry.
func getArray(fields mapstr.M, data []byte, offset uint32) (next uint32, err bool, exists bool) {
	length, err := getIntegerAt[uint32](data, offset)
	if err {
		logp.Debug("amqp", "Error while parsing a field table")
		return 0, true, false
	}
	offset64 := int64(offset)
	dataLen64 := int64(len(data))

	// less actual data than the transmitted length indicates
	if offset64 > dataLen64 || 4 > dataLen64-offset64 {
		return 0, true, false
	}
	arrayStart64 := offset64 + 4
	if int64(length) > dataLen64-arrayStart64 {
		return 0, true, false
	}
	if length > ^uint32(0)-offset-4 {
		return 0, true, false
	}
	if length > 0 {
		exists = true
		array := mapstr.M{}
		arrayStartInt := int(arrayStart64)
		arrayEndInt := arrayStartInt + int(length)
		err := fieldUnmarshal(array, data[arrayStartInt:arrayEndInt], 0, length, 0)
		if err {
			logp.Debug("amqp", "Error while parsing a field array")
			return 0, true, false
		}
		fields.Update(array)
	}
	return offset + 4 + length, false, exists
}

// The index parameter, when set at -1, indicates that the entry is a field table.
// If it's set at 0, it is an array.
func fieldUnmarshal(table mapstr.M, data []byte, offset uint32, length uint32, index int) (err bool) {
	var name string

	// Why is this returning false if attempting to offset past the length of the data?
	if offset >= length {
		return false
	}
	// get name of the field. If it's an array, it will be the index parameter as a
	// string. If it's a table, it will be the name of the field.
	if index < 0 {
		fieldName, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get short string in table")
			return true
		}
		name = fieldName
		offset += consumed
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
		v, err := getIntegerAt[int8](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get int8 in table")
			return true
		}
		table[name] = v
		offset += 2
	case shortShortUint:
		v, err := getIntegerAt[uint8](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get uint8 in table")
			return true
		}
		table[name] = v
		offset += 2
	case shortInt:
		v, err := getIntegerAt[int16](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get int16 in table")
			return true
		}
		table[name] = v
		offset += 3
	case shortUint:
		v, err := getIntegerAt[uint16](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get uint16 in table")
			return true
		}
		table[name] = v
		offset += 3
	case longInt:
		v, err := getIntegerAt[int32](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get int32 in table")
			return true
		}
		table[name] = v
		offset += 5
	case longUint:
		v, err := getIntegerAt[uint32](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get uint32 in table")
			return true
		}
		table[name] = v
		offset += 5
	case longLongInt:
		v, err := getIntegerAt[int64](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get int64 in table")
			return true
		}
		table[name] = v
		offset += 9
	case longLongUint:
		v, err := getIntegerAt[uint64](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get uint64 in table")
			return true
		}
		table[name] = v
		offset += 9
	case float:
		v, err := getIntegerAt[uint32](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get float32 in table")
			return true
		}
		table[name] = math.Float32frombits(v)
		offset += 5
	case double:
		v, err := getIntegerAt[uint64](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get float64 in table")
			return true
		}
		table[name] = math.Float64frombits(v)
		offset += 9
	case decimal:
		if len(data) < int(offset)+6 {
			logp.Debug("amqp", "Failed to get decimal in table")
			return true
		}
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
		s, consumed, err := getLVString[uint8](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get short string in table")
			return true
		}
		table[name] = s
		offset += consumed + 1
	case longString:
		s, consumed, err := getLVString[uint32](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get long string in table")
			return true
		}
		table[name] = s
		offset += consumed + 1
	case fieldArray:
		newMap := mapstr.M{}
		next, err, _ := getArray(newMap, data, offset+1)
		if err {
			return true
		}
		table[name] = newMap
		offset = next
	case timestamp:
		ts, err := getIntegerAt[int64](data, offset+1)
		if err {
			logp.Debug("amqp", "Failed to get timestamp in table")
			return true
		}
		t := time.Unix(ts, 0)
		table[name] = t.Format(amqpTimeLayout)
		offset += 9
	case fieldTable:
		newMap := mapstr.M{}
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
		size, err := getIntegerAt[uint32](data, offset+1)
		if err || len(data) < int(offset+5+size) {
			logp.Debug("amqp", "Failed to get byte array in table")
			return true
		}
		table[name] = bodyToByteArray(data[offset+5 : offset+5+size])
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
