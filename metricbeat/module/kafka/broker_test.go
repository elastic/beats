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

package kafka

import (
	"net"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type dummyNet struct{}

func (m *dummyNet) LookupIP(addr string) ([]net.IP, error) {
	dns := map[string][]net.IP{
		"kafka1": []net.IP{net.IPv4(10, 0, 0, 1)},
		"kafka2": []net.IP{net.IPv4(10, 0, 0, 2)},
		"kafka3": []net.IP{net.IPv4(10, 0, 0, 3)},
	}
	ips, found := dns[addr]
	if !found {
		return nil, errors.New("not found")
	}
	return ips, nil
}

func (m *dummyNet) LookupAddr(addr string) ([]string, error) {
	dns := map[string][]string{
		"10.0.0.1": []string{"kafka1"},
		"10.0.0.2": []string{"kafka2"},
		"10.0.0.3": []string{"kafka3"},
	}
	names, found := dns[addr]
	if !found {
		return nil, errors.New("not found")
	}
	return names, nil
}

func (m *dummyNet) LocalIPAddrs() ([]net.IP, error) {
	return []net.IP{
		net.IPv4(127, 0, 0, 1),
		net.IPv4(10, 0, 0, 2),
		net.IPv4(10, 1, 0, 2),
	}, nil
}

func (m *dummyNet) Hostname() (string, error) {
	return "kafka2", nil
}

func TestFindMatchingAddress(t *testing.T) {
	cases := []struct {
		title   string
		address string
		brokers []string
		index   int
		exists  bool
	}{
		{
			title:   "exists",
			address: "10.0.0.2:9092",
			brokers: []string{"10.0.0.1:9092", "10.0.0.2:9092"},
			index:   1,
			exists:  true,
		},
		{
			title:   "doesn't exist",
			address: "8.8.8.8:9092",
			brokers: []string{"10.0.0.1:9092", "10.0.0.2:9092"},
			exists:  false,
		},
		{
			title:   "exists on default port",
			address: "10.0.0.2",
			brokers: []string{"10.0.0.1:9092", "10.0.0.2:9092"},
			index:   1,
			exists:  true,
		},
		{
			title:   "multiple brokers on same host",
			address: "127.0.0.1:9093",
			brokers: []string{"127.0.0.1:9092", "127.0.0.1:9093", "127.0.0.1:9094"},
			index:   1,
			exists:  true,
		},
		{
			title:   "hostname",
			address: "kafka2:9092",
			brokers: []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
			index:   1,
			exists:  true,
		},
		{
			title:   "hostname and default port",
			address: "kafka2",
			brokers: []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
			index:   1,
			exists:  true,
		},
		{
			title:   "hostname and default port doesn't exist",
			address: "kafka2",
			brokers: []string{"kafka1:9092", "kafka2:9093", "kafka3:9092"},
			exists:  false,
		},
		{
			title:   "hostname with ip brokers",
			address: "kafka2:9092",
			brokers: []string{"10.0.0.1:9092", "10.0.0.2:9092", "10.0.0.3:9092"},
			index:   1,
			exists:  true,
		},
		{
			title:   "ip with named brokers",
			address: "10.0.0.2:9092",
			brokers: []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
			index:   1,
			exists:  true,
		},
		{
			title:   "ip with multiple local brokers without name",
			address: "10.1.0.2:9094",
			brokers: []string{"10.1.0.2:9092", "10.1.0.2:9093", "10.1.0.2:9094"},
			index:   2,
			exists:  true,
		},
	}

	finder := brokerFinder{Net: &dummyNet{}}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			i, found := finder.findAddress(c.address, c.brokers)
			if c.exists {
				if assert.True(t, found, "broker expected to be found") {
					assert.Equal(t, c.index, i, "incorrect broker match")
				}
			} else {
				assert.False(t, found, "broker shouldn't be found")
			}
		})
	}
}
