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

package netmetrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestAddrs(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		addr4, addr6, err := addrs("0.0.0.0:65535", logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(addr4) == 0 {
			t.Errorf("expected addr in addr4 for IPv4 address: addr6 is %v", addr6)
		}
		if len(addr6) != 0 {
			t.Errorf("unexpected addrs in addr6 for IPv4 address: %v", addr6)
		}
	})

	t.Run("ipv6", func(t *testing.T) {
		addr4, addr6, err := addrs("[::]:9001", logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(addr4) != 0 {
			t.Errorf("unexpected addr in addr4 for IPv6 address: %v", addr4)
		}
		if len(addr6) == 0 {
			t.Errorf("expected addrs in addr6 for IPv6 address: addr4 is %v", addr4)
		}
	})
}

func TestUDPMetrics(t *testing.T) {
	reg := monitoring.NewRegistry()
	m := NewUDP(reg, "localhost:4242", 1000, time.Second, logp.NewNopLogger())

	data := make([]byte, 42, 42)
	start := time.Now()
	now := start

	numEvents := 100
	for range 100 {
		now = start.Add(100 * time.Millisecond)
		m.EventReceived(len(data), now)
		m.EventPublished(now)
	}

	assert.EqualValues(t, len(data)*numEvents, m.bytes.Get(), "wrong value for bytes received")
	assert.EqualValues(t, numEvents, m.packets.Get(), "wrong value for packets received")
	assert.EqualValues(t, numEvents, m.processingTime.Count(), "wrong number of events for processing time calculation")
	assert.EqualValues(t, numEvents-1, m.arrivalPeriod.Count(), "wrong number of events for processing time calculation")
}

func TestTCPMetrics(t *testing.T) {
	reg := monitoring.NewRegistry()
	m := NewTCP(reg, "localhost:4242", time.Second, logp.NewNopLogger())

	data := make([]byte, 42, 42)
	start := time.Now()
	now := start

	numEvents := 100
	for range 100 {
		now = start.Add(100 * time.Millisecond)
		m.EventReceived(len(data), now)
		m.EventPublished(now)
	}

	assert.EqualValues(t, len(data)*numEvents, m.bytes.Get(), "wrong value for bytes received")
	assert.EqualValues(t, numEvents, m.packets.Get(), "wrong value for packets received")
	assert.EqualValues(t, numEvents, m.processingTime.Count(), "wrong number of events for processing time calculation")
	assert.EqualValues(t, numEvents-1, m.arrivalPeriod.Count(), "wrong number of events for processing time calculation")
}
