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

	"github.com/shirou/gopsutil/v4/net"
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
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "SYN_SENT",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "SYN_RECV",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "SYN_RECV",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LAST_ACK",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "FIN_WAIT1",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "FIN_WAIT2",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "FIN_WAIT2",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LAST_ACK",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSING",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSING",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "FIN_WAIT2",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LAST_ACK",
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

	tcpSynsents, err := metrics.GetValue("tcp.all.syn_sent")

	if err != nil {
		t.Fail()
	}

	tcpSynrecvs, err := metrics.GetValue("tcp.all.syn_recv")

	if err != nil {
		t.Fail()
	}

	tcpFinwait1s, err := metrics.GetValue("tcp.all.fin_wait1")

	if err != nil {
		t.Fail()
	}

	tcpFinwait2s, err := metrics.GetValue("tcp.all.fin_wait2")

	if err != nil {
		t.Fail()
	}

	tcpLastacks, err := metrics.GetValue("tcp.all.last_ack")

	if err != nil {
		t.Fail()
	}

	tcpClosings, err := metrics.GetValue("tcp.all.closing")

	if err != nil {
		t.Fail()
	}

	assert.Equal(t, 23, allConns)
	assert.Equal(t, 2, allListens)
	assert.Equal(t, 2, udpConns)
	assert.Equal(t, 21, tcpConns)
	assert.Equal(t, 2, tcpListens)
	assert.Equal(t, 2, tcpEstablisheds)
	assert.Equal(t, 3, tcpClosewaits)
	assert.Equal(t, 1, tcpTimewaits)
	assert.Equal(t, 1, tcpSynsents)
	assert.Equal(t, 2, tcpSynrecvs)
	assert.Equal(t, 1, tcpFinwait1s)
	assert.Equal(t, 3, tcpFinwait2s)
	assert.Equal(t, 3, tcpLastacks)
	assert.Equal(t, 2, tcpClosings)
}
