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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/input/internal/procnet"
)

func TestProcNetTCP(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		path := "testdata/proc_net_tcp.txt"
		t.Run("with_match", func(t *testing.T) {
			addr := []string{procnet.IPv4(net.IP{0x7f, 0x00, 0x00, 0x01}, 0x17ac)}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 1, rx)
		})

		t.Run("leading_zero", func(t *testing.T) {
			addr := []string{procnet.IPv4(net.IP{0x00, 0x7f, 0x01, 0x00}, 0x17af)}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 1, rx)
		})

		t.Run("unspecified", func(t *testing.T) {
			addr := []string{procnet.IPv4(net.ParseIP("0.0.0.0"), 0x17ac)}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 2, rx)
		})

		t.Run("without_match", func(t *testing.T) {
			addr := []string{
				procnet.IPv4(net.IP{0xde, 0xad, 0xbe, 0xef}, 0xf00d),
				procnet.IPv4(net.IP{0xba, 0x1d, 0xfa, 0xce}, 0x1135),
			}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.Nil(t, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})

		t.Run("bad_addrs", func(t *testing.T) {
			addr := []string{"FOO:BAR", "BAR:BAZ"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.EqualValues(t, addr, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})
	})

	t.Run("IPv6", func(t *testing.T) {
		path := "testdata/proc_net_tcp6.txt"
		t.Run("with_match", func(t *testing.T) {
			addr := []string{procnet.IPv6(net.IP{0: 0x7f, 3: 0x01, 15: 0}, 0x17ac)}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 1, rx)
		})

		t.Run("leading_zero", func(t *testing.T) {
			addr := []string{procnet.IPv6(net.IP{1: 0x7f, 2: 0x01, 15: 0}, 0x17af)}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 1, rx)
		})

		t.Run("unspecified", func(t *testing.T) {
			addr := []string{procnet.IPv6(net.ParseIP("[::]"), 0x17ac)}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			rx, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(t, bad)
			assert.EqualValues(t, 2, rx)
		})

		t.Run("without_match", func(t *testing.T) {
			addr := []string{
				procnet.IPv6(net.IP{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}, 0xf00d),
				procnet.IPv6(net.IP{0xba, 0x1d, 0xfa, 0xce, 0xba, 0x1d, 0xfa, 0xce, 0xba, 0x1d, 0xfa, 0xce, 0xba, 0x1d, 0xfa, 0xce}, 0x1135),
			}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.Nil(t, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})

		t.Run("bad_addrs", func(t *testing.T) {
			addr := []string{"FOO:BAR", "BAR:BAZ"}
			hasUnspecified, addrIsUnspecified, bad := containsUnspecifiedAddr(addr)
			_, err := procNetTCP(path, addr, hasUnspecified, addrIsUnspecified)
			assert.EqualValues(t, addr, bad)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "entry not found")
			}
		})
	})
}

func FuzzProcNetTCP(f *testing.F) {
	path := "testdata/proc_net_tcp.txt"

	f.Add("0100007f:17AC")
	f.Fuzz(func(t *testing.T, addr string) {
		rx, err := procNetTCP(path, []string{addr}, false, []bool{false})
		if err == nil {
			// If err is nil, we hit one of the IPs on testdata/proc_net_tcp.txt
			// so we can check the value of rx.
			if rx != 1 {
				t.Errorf("expecting rx to be 1, got: %d", rx)
			}
		}
	})
}
