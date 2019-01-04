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

// +build windows

package procs

import (
	"encoding/hex"
	"fmt"
	"net"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestParseTableRaw(t *testing.T) {
	IPv4 := extractTCPRowOwnerPID
	IPv6 := extractTCP6RowOwnerPID

	for idx, testCase := range []struct {
		name     string
		factory  extractorFactory
		raw      string
		expected []portProcMapping
		mustErr  bool
	}{
		{"Empty table IPv4", IPv4,
			"00000000", nil, false},
		{"Empty table IPv6", IPv6,
			"00000000", nil, false},
		{"Short table (no length)", IPv4,
			"000000", nil, true},
		{"Short table (partial entry)", IPv6,
			"01000000AAAAAAAAAAAAAAAAAAAA", nil, true},
		{"One entry (IPv4)", IPv4,
			"01000000" +
				"77777777AAAAAAAA12340000BBBBBBBBFFFF0000CCCCCCCC",
			[]portProcMapping{
				{endpoint: endpoint{address: "170.170.170.170", port: 0x1234}, pid: 0xCCCCCCCC},
			}, false},
		{"Two entries (IPv6)", IPv6,
			"02000000" +
				// First entry
				"11112222333344445555666677778888F0F0F0F0" +
				"ABCDEFFF" + // local port
				"FFFFEEEEDDDDCCCCBBBBAAAA999988880A0A0A0A" +
				"33333333" + // remote port
				"77777777" +
				"01000000" + // pid
				// second entry
				"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABBBBBBBB" +
				"0000FFFF" + // local port
				"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBCCCCCCCC" +
				"44444444" + // remote port
				"77777777" +
				"FFFF0000" + // pid
				"",
			[]portProcMapping{
				{endpoint: endpoint{address: "1111:2222:3333:4444:5555:6666:7777:8888", port: 0xABCD}, pid: 1},
				{endpoint: endpoint{address: "aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa:aaaa", port: 0}, pid: 0xffff},
			}, false},
	} {
		msg := fmt.Sprintf("Test case #%d: %s", idx+1, testCase.name)
		table, err := hex.DecodeString(testCase.raw)
		assert.NoError(t, err, msg)
		var result []portProcMapping
		callback := func(ip net.IP, port uint16, pid int) {
			result = append(result, portProcMapping{endpoint: endpoint{ip.String(), port}, pid: pid})
		}
		err = parseTable(table, testCase.factory(callback))
		if testCase.mustErr {
			assert.Error(t, err, msg)
		} else {
			assert.NoError(t, err, msg)
			assert.Len(t, result, len(testCase.expected), msg)
			assert.Equal(t, testCase.expected, result, msg)
		}
	}
}

func TestParseTableSizes(t *testing.T) {
	// Make sure the structs in Golang have the expected size
	assert.Equal(t, uintptr(sizeOfTCPRowOwnerPID), unsafe.Sizeof(TCPRowOwnerPID{}))
	assert.Equal(t, uintptr(sizeOfTCP6RowOwnerPID), unsafe.Sizeof(TCP6RowOwnerPID{}))
}
