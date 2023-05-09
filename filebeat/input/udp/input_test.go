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

package udp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcNetUDP(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		path := "testdata/proc_net_udp.txt"
		t.Run("with_match", func(t *testing.T) {
			addr := []string{"2508640A:1BBE"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, drops, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 1, rx)
			assert.EqualValues(t, 2, drops)
		})

		t.Run("unspecified", func(t *testing.T) {
			addr := []string{"00000000:1BBE"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, drops, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 2, rx)
			assert.EqualValues(t, 4, drops)
		})

		t.Run("without_match", func(t *testing.T) {
			addr := []string{"deadbeef:f00d", "ba1dface:1135"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, _, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.Nil(t, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})

		t.Run("bad_addrs", func(t *testing.T) {
			addr := []string{"FOO:BAR", "BAR:BAZ"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, _, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.EqualValues(t, addr, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})
	})

	t.Run("IPv6", func(t *testing.T) {
		path := "testdata/proc_net_udp6.txt"
		t.Run("with_match", func(t *testing.T) {
			addr := []string{"0000000000000000000000000100007f:1BBD"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, drops, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 1, rx)
			assert.EqualValues(t, 475174, drops)
		})

		t.Run("unspecified", func(t *testing.T) {
			addr := []string{"00000000000000000000000000000000:1BBD"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, drops, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 2, rx)
			assert.EqualValues(t, 2*475174, drops)
		})

		t.Run("without_match", func(t *testing.T) {
			addr := []string{"deadbeefdeadbeefdeadbeefdeadbeef:f00d", "ba1dfaceba1dfaceba1dfaceba1dface:1135"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, _, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.Nil(t, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})

		t.Run("bad_addrs", func(t *testing.T) {
			addr := []string{"FOO:BAR", "BAR:BAZ"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, _, err := procNetUDP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.EqualValues(t, addr, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})
	})
}
