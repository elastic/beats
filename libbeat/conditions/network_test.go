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

package conditions

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNetworkConfigUnpack(t *testing.T) {
	testYAMLConfig := func(t *testing.T, expected bool, evt *beat.Event, yml string) {
		c, err := common.NewConfigWithYAML([]byte(yml), "test")
		if err != nil {
			t.Fatal(err)
		}

		var config Config
		if err = c.Unpack(&config); err != nil {
			t.Fatal(err)
		}

		testConfig(t, expected, evt, &config)
	}

	t.Run("string values", func(t *testing.T) {
		const yaml = `
network:
  client_ip: loopback
  ip: loopback
`
		testYAMLConfig(t, true, httpResponseTestEvent, yaml)
	})

	t.Run("array values", func(t *testing.T) {
		const yaml = `
network:
  client_ip: [loopback]
  ip: [loopback]
`
		testYAMLConfig(t, true, httpResponseTestEvent, yaml)
	})

	t.Run("nested keys", func(t *testing.T) {
		const yaml = `
network:
  ip:
    client: [loopback]
    server: [loopback]
`

		evt := &beat.Event{Fields: mapstr.M{
			"ip": mapstr.M{
				"client": "127.0.0.1",
				"server": "127.0.0.1",
			},
		}}

		testYAMLConfig(t, true, evt, yaml)
	})
}

func TestNetworkCreate(t *testing.T) {
	t.Run("all options", func(t *testing.T) {
		c, err := NewCondition(&Config{
			Network: map[string]interface{}{
				"ipv4_ip":                      "192.168.10.1/16",
				"ipv6_ip":                      "fd00::/8",
				"loopback_ip":                  "loopback",
				"unicast_ip":                   "unicast",
				"global_unicast_ip":            "global_unicast",
				"link_local_unicast_ip":        "link_local_unicast",
				"interface_local_multicast_ip": "interface_local_multicast",
				"link_local_multicast_ip":      "link_local_multicast",
				"multicast_ip":                 "multicast",
				"unspecified_ip":               "unspecified",
				"private_ip":                   "private",
				"public_ip":                    "public",
			},
		})
		if assert.NoError(t, err) {
			t.Log(c)
		}
	})

	t.Run("invalid keyword", func(t *testing.T) {
		_, err := NewCondition(&Config{
			Network: map[string]interface{}{
				"invalid": "loop-back",
			},
		})
		assert.Error(t, err)
	})

	t.Run("bad cidr", func(t *testing.T) {
		_, err := NewCondition(&Config{
			Network: map[string]interface{}{
				"bad_cidr": "127.0/8",
			},
		})
		assert.Error(t, err)
	})

	t.Run("bad type", func(t *testing.T) {
		_, err := NewCondition(&Config{
			Network: map[string]interface{}{
				"bad_type": 1,
			},
		})
		assert.Error(t, err)
	})
}

func TestNetworkCheck(t *testing.T) {
	t.Run("match loopback", func(t *testing.T) {
		testConfig(t, true, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"ip": "127.0.0.0/8",
			},
		})
	})

	t.Run("negative match", func(t *testing.T) {
		testConfig(t, false, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"ip": "192.168.0.0/16",
			},
		})
	})

	t.Run("wrong field value type", func(t *testing.T) {
		testConfig(t, false, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"status": "unicast",
			},
		})
	})

	t.Run("multiple fields match", func(t *testing.T) {
		testConfig(t, true, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"client_ip": "loopback",
				"ip":        "127.0.0.0/24",
			},
		})
	})

	// Multiple conditions are treated as an implicit AND.
	t.Run("multiple fields negative match", func(t *testing.T) {
		testConfig(t, false, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"client_ip": "multicast",
				"ip":        "127.0.0.0/24",
			},
		})
	})

	t.Run("field not present", func(t *testing.T) {
		testConfig(t, false, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"does_not_exist": "multicast",
			},
		})
	})

	t.Run("multiple values match", func(t *testing.T) {
		testConfig(t, true, httpResponseTestEvent, &Config{
			Network: map[string]interface{}{
				"client_ip": []interface{}{"public", "loopback"},
			},
		})
	})
}

func TestNetworkPrivate(t *testing.T) {
	t.Run("ranges", func(t *testing.T) {
		var equal = func(cidr string, actual net.IPNet) {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				t.Fatal(err)
			}

			assert.True(t, network.IP.Equal(actual.IP))
			assert.EqualValues(t, network.Mask, actual.Mask)
		}

		equal("fd00::/8", privateIPv6)
		equal("10.0.0.0/8", privateIPv4[0])
		equal("172.16.0.0/12", privateIPv4[1])
		equal("192.168.0.0/16", privateIPv4[2])
	})

	t.Run("match", func(t *testing.T) {
		isPrivate := func(ip string) { assert.True(t, isPrivateNetwork(net.ParseIP(ip)), "%v", ip) }
		isPrivate("10.0.0.0")
		isPrivate("10.255.255.255")
		isPrivate("192.168.0.0")
		isPrivate("192.168.255.255")
		isPrivate("172.16.0.0")
		isPrivate("172.31.255.255")
		isPrivate("fd11:3456:789a:1::1")

		isNotPrivate := func(ip string) { assert.False(t, isPrivateNetwork(net.ParseIP(ip)), "%v", ip) }
		isNotPrivate("192.0.2.1")
		isNotPrivate("2001:db8:ffff:ffff:ffff:ffff:ffff:1")
	})
}

func TestNetworkContains(t *testing.T) {
	ip := net.ParseIP("192.168.0.1")

	contains, err := NetworkContains(ip, "192.168.1.0/24", "192.168.0.0/24")
	assert.NoError(t, err)
	assert.True(t, contains)

	contains, err = NetworkContains(ip, "192.168.1.1", "192.168.0.0/24")
	assert.Error(t, err)
	assert.False(t, contains)

	// The second network is invalid but we don't validate them upfront.
	contains, err = NetworkContains(ip, "192.168.0.0/24", "192.168.1.1")
	assert.NoError(t, err)
	assert.True(t, contains)
}

func BenchmarkNetworkCondition(b *testing.B) {
	c, err := NewCondition(&Config{
		Network: map[string]interface{}{
			"ip": "192.168.0.1/16",
		},
	})
	if err != nil {
		b.Fatal(err)
	}

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"@timestamp": "2015-06-11T09:51:23.642Z",
			"ip":         "192.168.0.92",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Check(event)
	}
}
