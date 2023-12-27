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

//go:build windows

package procs

import (
	"bytes"
	"encoding/binary"
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

	pid := uint32(0xCCCCCCCC)
	for idx, testCase := range []struct {
		name     string
		factory  func(fn callbackFn) extractor
		raw      string
		expected []portProcMapping
		mustErr  bool
	}{
		{
			"Empty table IPv4", IPv4,
			"00000000", nil, false,
		},
		{
			"Empty table IPv6", IPv6,
			"00000000", nil, false,
		},
		{
			"Short table (no length)", IPv4,
			"000000", nil, true,
		},
		{
			"Short table (partial entry)", IPv6,
			"01000000AAAAAAAAAAAAAAAAAAAA", nil, true,
		},
		{
			"One entry (IPv4)", IPv4,
			"01000000" +
				"77777777AAAAAAAA12340000BBBBBBBBFFFF0000CCCCCCCC",
			[]portProcMapping{
				{endpoint: endpoint{address: "170.170.170.170", port: 0x1234}, pid: int(pid)},
			},
			false,
		},
		{
			"Two entries (IPv6)", IPv6,
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
			},
			false,
		},
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

func TestAddressIPv4(t *testing.T) {
	// The dwLocalAddr and dwRemoteAddr members are stored as a DWORD in the same format as the in_addr structure.
	// e.g. https://docs.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcprow_owner_pid#remarks
	network := binary.BigEndian

	for _, test := range []struct {
		// https://docs.microsoft.com/en-us/windows/win32/api/winsock2/ns-winsock2-in_addr
		a, b, c, d uint8
	}{
		{a: 1, b: 2, c: 3, d: 4},
		{a: 128, b: 64, c: 196, d: 32},
	} {
		var buf bytes.Buffer
		err := binary.Write(&buf, network, test)
		if err != nil {
			t.Errorf("failed to write %+v: %v", test, err)
			continue
		}
		dword := *(*uint32)(unsafe.Pointer((*[4]byte)(buf.Bytes())))
		got := addressIPv4(dword)
		want := net.IP{test.a, test.b, test.c, test.d}
		if !got.Equal(want) {
			t.Errorf("unexpected result from %+v: got:%d want:%d", test, got, want)
		}
	}
}

func TestUint32FieldToPort(t *testing.T) {
	// The dwLocalPort, and dwRemotePort members are in network byte order.
	// e.g. https://docs.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcprow_owner_pid#remarks
	network := binary.BigEndian

	for _, test := range []struct {
		port  uint16
		decoy uint16
	}{
		{port: 1, decoy: 0xffff},
		{port: 2, decoy: 0xffff},
		{port: 128, decoy: 0xffff},
		{port: 256, decoy: 0xffff},
		{port: 512, decoy: 0xffff},
		{port: 512, decoy: 0xffff},
		{port: 32767, decoy: 0xffff},
	} {
		var buf bytes.Buffer
		err := binary.Write(&buf, network, test)
		if err != nil {
			t.Errorf("failed to write %+v: %v", test, err)
			continue
		}
		dword0 := *(*uint32)(unsafe.Pointer((*[4]byte)(buf.Bytes())))
		got := uint32FieldToPort(dword0)
		want := test.port
		if got != want {
			t.Errorf("unexpected result from %+v: got:%d want:%d", test, got, want)
		}
	}
}
