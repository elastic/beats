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

package socket

import (
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListenerTable(t *testing.T) {
	l := NewListenerTable()

	proto := uint8(4)
	lAddr := net.ParseIP("192.0.2.1")
	httpPort := 80
	rAddr := net.ParseIP("198.18.0.1")
	ephemeralPort := 48199
	ipv6Addr := net.ParseIP("2001:db8:fe80::217:f2ff:fe07:ed62")
	ipv4InIpv6 := net.ParseIP("::ffff:127.0.0.1")

	// Any socket with remote port of 0 is listening.
	assert.Equal(t, Listening, l.Direction(syscall.AF_INET, proto, lAddr, httpPort, net.IPv4zero, 0))

	// Listener on 192.0.2.1:80
	l.Put(proto, lAddr, httpPort)

	assert.Equal(t, Inbound, l.Direction(syscall.AF_INET, proto, lAddr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outbound, l.Direction(syscall.AF_INET, 0, lAddr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outbound, l.Direction(syscall.AF_INET, proto, lAddr, ephemeralPort, rAddr, ephemeralPort))

	// Listener on 0.0.0.0:80
	l.Reset()
	l.Put(proto, net.IPv4zero, httpPort)

	assert.Equal(t, Inbound, l.Direction(syscall.AF_INET, proto, lAddr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outbound, l.Direction(syscall.AF_INET6, proto, ipv6Addr, httpPort, rAddr, ephemeralPort))

	// Listener on :::80
	l.Reset()
	l.Put(proto, net.IPv6zero, httpPort)

	assert.Equal(t, Inbound, l.Direction(syscall.AF_INET6, proto, ipv6Addr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Inbound, l.Direction(syscall.AF_INET6, proto, ipv4InIpv6, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outbound, l.Direction(syscall.AF_INET, proto, lAddr, httpPort, rAddr, ephemeralPort))
}
