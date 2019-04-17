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

package socket_summary

import (
	"syscall"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"
)

func getMockedConns() []net.ConnectionStat {
	return []net.ConnectionStat{
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_DGRAM,
			Status: "",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_DGRAM,
			Status: "",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LISTEN",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "ESTABLISHED",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "ESTABLISHED",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSE_WAIT",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "TIME_WAIT",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSE_WAIT",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSE_WAIT",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSE",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LISTEN",
		},
	}
}

func TestCalculateConnStats(t *testing.T) {
	conns := getMockedConns()
	metrics := calculateConnStats(conns)

	allConns, err := metrics.GetValue("all.count")

	if err != nil {
		t.Fail()
	}

	allListens, err := metrics.GetValue("all.listening")

	if err != nil {
		t.Fail()
	}

	udpConns, err := metrics.GetValue("udp.all.count")

	if err != nil {
		t.Fail()
	}

	tcpConns, err := metrics.GetValue("tcp.all.count")

	if err != nil {
		t.Fail()
	}

	tcpListens, err := metrics.GetValue("tcp.all.listening")

	if err != nil {
		t.Fail()
	}

	tcpEstablisheds, err := metrics.GetValue("tcp.all.established")

	if err != nil {
		t.Fail()
	}

	tcpClosewaits, err := metrics.GetValue("tcp.all.close_wait")

	if err != nil {
		t.Fail()
	}

	tcpTimewaits, err := metrics.GetValue("tcp.all.time_wait")

	if err != nil {
		t.Fail()
	}

	assert.Equal(t, allConns, 11)
	assert.Equal(t, allListens, 2)
	assert.Equal(t, udpConns, 2)
	assert.Equal(t, tcpConns, 9)
	assert.Equal(t, tcpListens, 2)
	assert.Equal(t, tcpEstablisheds, 2)
	assert.Equal(t, tcpClosewaits, 3)
	assert.Equal(t, tcpTimewaits, 1)
}
