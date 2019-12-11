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
)

// Direction indicates how a socket was initiated.
type Direction uint8

const (
	_ Direction = iota
	// Inbound indicates a connection was established from the outside to
	// listening socket on this host.
	Inbound
	// Outbound indicates a connection was established from this socket to an
	// external listening socket.
	Outbound
	// Listening indicates a socket that is listening.
	Listening
)

// Names for the direction of a connection
const (
	InboundName   = "inbound"
	OutboundName  = "outbound"
	ListeningName = "listening"
)

var directionNames = map[Direction]string{
	Inbound:   InboundName,
	Outbound:  OutboundName,
	Listening: ListeningName,
}

func (d Direction) String() string {
	if name, exists := directionNames[d]; exists {
		return name
	}
	return "unknown"
}

// ipList is a list of IP addresses.
type ipList struct {
	ips []net.IP
}

func (l *ipList) put(ip net.IP) { l.ips = append(l.ips, ip) }

// portTable is a mapping of port number to listening IP addresses.
type portTable map[int]*ipList

// protocolTable is a mapping of protocol numbers to listening ports.
type protocolTable map[uint8]portTable

// ListenerTable tracks sockets that are listening. It can then be used to
// identify if a socket is listening, incoming, or outgoing.
type ListenerTable struct {
	data protocolTable
}

// NewListenerTable returns a new ListenerTable.
func NewListenerTable() *ListenerTable {
	return &ListenerTable{
		data: protocolTable{},
	}
}

// Reset resets all data in the table.
func (t *ListenerTable) Reset() {
	for _, ports := range t.data {
		for port := range ports {
			delete(ports, port)
		}
	}
}

// Put puts a new listening address into the table.
func (t *ListenerTable) Put(proto uint8, ip net.IP, port int) {
	ports, exists := t.data[proto]
	if !exists {
		ports = portTable{}
		t.data[proto] = ports
	}

	// Add port + addr to table.
	interfaces, exists := ports[port]
	if !exists {
		interfaces = &ipList{}
		ports[port] = interfaces
	}
	interfaces.put(ip)
}

// Direction returns whether the connection was incoming or outgoing based on
// the protocol and local address. It compares the given local address to the
// listeners in the table for the protocol and returns Inbound if there is a
// match. If remotePort is 0 then Listening is returned.
func (t *ListenerTable) Direction(
	family uint8, proto uint8,
	localIP net.IP, localPort int,
	remoteIP net.IP, remotePort int,
) Direction {
	if remotePort == 0 {
		return Listening
	}

	// Are there any listeners on the given protocol?
	ports, exists := t.data[proto]
	if !exists {
		return Outbound
	}

	// Is there any listener on the port?
	interfaces, exists := ports[localPort]
	if !exists {
		return Outbound
	}

	// Is there a listener that specific interface? OR
	// Is there a listener on the "any" address (0.0.0.0 or ::)?
	for _, ip := range interfaces.ips {
		switch {
		case ip.Equal(localIP):
			return Inbound
		case family == syscall.AF_INET && ip.Equal(net.IPv4zero):
			return Inbound
		case family == syscall.AF_INET6 && ip.Equal(net.IPv6zero):
			return Inbound
		}
	}

	return Outbound
}
