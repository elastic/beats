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

//go:build !requirefips

package translate_ldap_attribute

import (
	"net"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLogonServerPreservesHostname(t *testing.T) {
	t.Setenv("LOGONSERVER", "\\\\DC01")

	originalResolver := resolveTCPAddr
	resolveTCPAddr = func(network, address string) (*net.TCPAddr, error) {
		return &net.TCPAddr{IP: net.ParseIP("192.0.2.10")}, nil
	}
	t.Cleanup(func() { resolveTCPAddr = originalResolver })

	log := logp.NewLogger("test")
	addresses := findLogonServer(true, log)
	require.Len(t, addresses, 2)
	assert.Equal(t, "ldaps://DC01:636", addresses[0])
	assert.Equal(t, "ldaps://192.0.2.10:636", addresses[1])
}

func TestFindLogonServerFallsBackWithoutResolution(t *testing.T) {
	t.Setenv("LOGONSERVER", "\\\\DC02")

	originalResolver := resolveTCPAddr
	resolveTCPAddr = func(network, address string) (*net.TCPAddr, error) {
		return nil, assert.AnError
	}
	t.Cleanup(func() { resolveTCPAddr = originalResolver })

	log := logp.NewLogger("test")
	addresses := findLogonServer(false, log)
	require.Len(t, addresses, 1)
	assert.Equal(t, "ldap://DC02:389", addresses[0])
}

type fakeRand struct {
	values []int
}

func (f *fakeRand) Intn(n int) int {
	if len(f.values) == 0 {
		return 0
	}
	v := f.values[0]
	f.values = f.values[1:]
	if n <= 0 {
		return 0
	}
	if v < 0 {
		v = 0
	}
	if v >= n {
		v = n - 1
	}
	return v
}

func TestOrderSRVRecordsPriorityAndWeight(t *testing.T) {
	records := []*net.SRV{
		{Target: "low1.example.com.", Port: 389, Priority: 10, Weight: 1},
		{Target: "low2.example.com.", Port: 389, Priority: 10, Weight: 1},
		{Target: "heavy.example.com.", Port: 389, Priority: 10, Weight: 100},
		{Target: "high.example.com.", Port: 389, Priority: 5, Weight: 1},
	}

	r := &fakeRand{values: []int{0, 101, 0, 0}}
	ordered := orderSRVRecords(records, r)

	require.Len(t, ordered, len(records))
	assert.Equal(t, "high.example.com.", ordered[0].Target)
	assert.Equal(t, "heavy.example.com.", ordered[1].Target)
}
