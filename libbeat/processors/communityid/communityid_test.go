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

package communityid

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestNewDefaults(t *testing.T) {
	_, err := New(common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	// From flowhash package testdata.
	// 1:LQU9qZlK+B5F3KDmev6m5PMibrg= | 128.232.110.120 66.35.250.204 6 34855 80
	evt := func() common.MapStr {
		return common.MapStr{
			"source": common.MapStr{
				"ip":   "128.232.110.120",
				"port": 34855,
			},
			"destination": common.MapStr{
				"ip":   "66.35.250.204",
				"port": 80,
			},
			"network": common.MapStr{"transport": "TCP"},
		}
	}

	t.Run("valid", func(t *testing.T) {
		testProcessor(t, 0, evt(), "1:LQU9qZlK+B5F3KDmev6m5PMibrg=")
	})

	t.Run("seed", func(t *testing.T) {
		testProcessor(t, 123, evt(), "1:hTSGlFQnR58UCk+NfKRZzA32dPg=")
	})

	t.Run("invalid source IP", func(t *testing.T) {
		e := evt()
		e.Put("source.ip", 2162716280)
		testProcessor(t, 0, e, nil)
	})

	t.Run("invalid source port", func(t *testing.T) {
		e := evt()
		e.Put("source.port", 0)
		testProcessor(t, 0, e, nil)
	})

	t.Run("invalid destination IP", func(t *testing.T) {
		e := evt()
		e.Put("destination.ip", "308.111.1.2.3")
		testProcessor(t, 0, e, nil)
	})

	t.Run("invalid destination port", func(t *testing.T) {
		e := evt()
		e.Put("source.port", nil)
		testProcessor(t, 0, e, nil)
	})

	t.Run("unknown protocol", func(t *testing.T) {
		e := evt()
		e.Put("network.transport", "xyz")
		testProcessor(t, 0, e, nil)
	})

	t.Run("icmp", func(t *testing.T) {
		e := evt()
		e.Put("network.transport", "icmp")
		e.Put("icmp.type", 3)
		e.Put("icmp.code", 3)
		testProcessor(t, 0, e, "1:KF3iG9XD24nhlSy4r1TcYIr5mfE=")
	})

	t.Run("icmp without typecode", func(t *testing.T) {
		// Hashes src_ip + dst_ip + protocol with zero value typecode.
		e := evt()
		e.Put("network.transport", "icmp")
		testProcessor(t, 0, e, "1:PAE85ZfR4SbNXl5URZwWYyDehwU=")
	})

	t.Run("igmp", func(t *testing.T) {
		e := evt()
		e.Delete("source.port")
		e.Delete("destination.port")
		e.Put("network.transport", "igmp")
		testProcessor(t, 0, e, "1:D3t8Q1aFA6Ev0A/AO4i9PnU3AeI=")
	})

	t.Run("protocol number as string", func(t *testing.T) {
		e := evt()
		e.Delete("source.port")
		e.Delete("destination.port")
		e.Put("network.transport", "2")
		testProcessor(t, 0, e, "1:D3t8Q1aFA6Ev0A/AO4i9PnU3AeI=")
	})

	t.Run("protocol number", func(t *testing.T) {
		e := evt()
		e.Delete("source.port")
		e.Delete("destination.port")
		e.Put("network.transport", 2)
		testProcessor(t, 0, e, "1:D3t8Q1aFA6Ev0A/AO4i9PnU3AeI=")
	})
}

func testProcessor(t testing.TB, seed uint16, fields common.MapStr, expectedHash interface{}) {
	t.Helper()

	c := defaultConfig()
	c.Seed = seed
	p, err := newFromConfig(c)
	if err != nil {
		t.Fatal(err)
	}

	out, err := p.Run(&beat.Event{Fields: fields})
	if err != nil {
		t.Fatal(err)
	}

	id, _ := out.GetValue(c.Target)
	assert.EqualValues(t, expectedHash, id)
}
