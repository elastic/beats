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

package tcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcNetTCP(t *testing.T) {
	t.Run("with_match", func(t *testing.T) {
		addr := []string{"0100007F:17AC"}
		hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
		rx, err := procNetTCP("testdata/proc_net_tcp.txt", addr, hasUnspecified, addrIsUnspecified)
		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, bad)
		assert.EqualValues(t, 1, rx)
	})

	t.Run("unspecified", func(t *testing.T) {
		addr := []string{"00000000:17AC"}
		hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
		rx, err := procNetTCP("testdata/proc_net_tcp.txt", addr, hasUnspecified, addrIsUnspecified)
		if err != nil {
			t.Fatal(err)
		}
		assert.Nil(t, bad)
		assert.EqualValues(t, 2, rx)
	})

	t.Run("without_match", func(t *testing.T) {
		addr := []string{"deadbeef:f00d", "ba1dface:1135"}
		hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
		_, err := procNetTCP("testdata/proc_net_tcp.txt", addr, hasUnspecified, addrIsUnspecified)
		assert.Nil(t, bad)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "entry not found")
		}
	})

	t.Run("bad_addrs", func(t *testing.T) {
		addr := []string{"FOO:BAR", "BAR:BAZ"}
		hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
		_, err := procNetTCP("testdata/proc_net_tcp.txt", addr, hasUnspecified, addrIsUnspecified)
		assert.EqualValues(t, addr, bad)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "entry not found")
		}
	})
}
