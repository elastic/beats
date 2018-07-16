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

package common

import (
	"fmt"
	"net"
)

// In order for the IpPortTuple and the TcpTuple to be used as
// hashtable keys, they need to have a fixed size. This means the
// net.IP is problematic because it's internally represented as a slice.
// We're introducing the HashableIpPortTuple and the HashableTcpTuple
// types which are internally simple byte arrays.

const MaxIPPortTupleRawSize = 16 + 16 + 2 + 2

type HashableIPPortTuple [MaxIPPortTupleRawSize]byte

type BaseTuple struct {
	SrcIP, DstIP     net.IP
	SrcPort, DstPort uint16
}

type IPPortTuple struct {
	BaseTuple

	IPLength int

	raw    HashableIPPortTuple // Src_ip:Src_port:Dst_ip:Dst_port
	revRaw HashableIPPortTuple // Dst_ip:Dst_port:Src_ip:Src_port
}

func NewIPPortTuple(ipLength int, srcIP net.IP, srcPort uint16,
	dstIP net.IP, dstPort uint16) IPPortTuple {

	tuple := IPPortTuple{
		IPLength: ipLength,
		BaseTuple: BaseTuple{
			SrcIP:   srcIP,
			DstIP:   dstIP,
			SrcPort: srcPort,
			DstPort: dstPort,
		},
	}
	tuple.ComputeHashables()

	return tuple
}

func (t *IPPortTuple) ComputeHashables() {
	copy(t.raw[0:16], t.SrcIP)
	copy(t.raw[16:18], []byte{byte(t.SrcPort >> 8), byte(t.SrcPort)})
	copy(t.raw[18:34], t.DstIP)
	copy(t.raw[34:36], []byte{byte(t.DstPort >> 8), byte(t.DstPort)})

	copy(t.revRaw[0:16], t.DstIP)
	copy(t.revRaw[16:18], []byte{byte(t.DstPort >> 8), byte(t.DstPort)})
	copy(t.revRaw[18:34], t.SrcIP)
	copy(t.revRaw[34:36], []byte{byte(t.SrcPort >> 8), byte(t.SrcPort)})
}

func (t *IPPortTuple) String() string {
	return fmt.Sprintf("IpPortTuple src[%s:%d] dst[%s:%d]",
		t.SrcIP.String(),
		t.SrcPort,
		t.DstIP.String(),
		t.DstPort)
}

// Hashable returns a hashable value that uniquely identifies
// the IP-port tuple.
func (t *IPPortTuple) Hashable() HashableIPPortTuple {
	return t.raw
}

// Hashable returns a hashable value that uniquely identifies
// the IP-port tuple after swapping the source and destination.
func (t *IPPortTuple) RevHashable() HashableIPPortTuple {
	return t.revRaw
}

const MaxTCPTupleRawSize = 16 + 16 + 2 + 2 + 4

type HashableTCPTuple [MaxTCPTupleRawSize]byte

type TCPTuple struct {
	BaseTuple
	IPLength int

	StreamID uint32

	raw HashableTCPTuple // Src_ip:Src_port:Dst_ip:Dst_port:stream_id
}

func TCPTupleFromIPPort(t *IPPortTuple, streamID uint32) TCPTuple {
	tuple := TCPTuple{
		IPLength: t.IPLength,
		BaseTuple: BaseTuple{
			SrcIP:   t.SrcIP,
			DstIP:   t.DstIP,
			SrcPort: t.SrcPort,
			DstPort: t.DstPort,
		},
		StreamID: streamID,
	}
	tuple.ComputeHashables()

	return tuple
}

func (t *TCPTuple) ComputeHashables() {
	copy(t.raw[0:16], t.SrcIP)
	copy(t.raw[16:18], []byte{byte(t.SrcPort >> 8), byte(t.SrcPort)})
	copy(t.raw[18:34], t.DstIP)
	copy(t.raw[34:36], []byte{byte(t.DstPort >> 8), byte(t.DstPort)})
	copy(t.raw[36:40], []byte{byte(t.StreamID >> 24), byte(t.StreamID >> 16),
		byte(t.StreamID >> 8), byte(t.StreamID)})
}

func (t TCPTuple) String() string {
	return fmt.Sprintf("TcpTuple src[%s:%d] dst[%s:%d] stream_id[%d]",
		t.SrcIP.String(),
		t.SrcPort,
		t.DstIP.String(),
		t.DstPort,
		t.StreamID)
}

// Returns a pointer to the equivalent IpPortTuple.
func (t TCPTuple) IPPort() *IPPortTuple {
	ipport := NewIPPortTuple(t.IPLength, t.SrcIP, t.SrcPort,
		t.DstIP, t.DstPort)
	return &ipport
}

// Hashable() returns a hashable value that uniquely identifies
// the TCP tuple.
func (t *TCPTuple) Hashable() HashableTCPTuple {
	return t.raw
}

// CmdlineTuple contains the source and destination process names, as found by
// the proc module.
type CmdlineTuple struct {
	// Source and destination processes names as specified in packetbeat.procs.monitored
	Src, Dst []byte
	// Source and destination full command lines
	SrcCommand, DstCommand []byte
}

// Reverse returns a copy of the receiver with the source and destination fields
// swapped.
func (c *CmdlineTuple) Reverse() CmdlineTuple {
	return CmdlineTuple{
		Src:        c.Dst,
		Dst:        c.Src,
		SrcCommand: c.DstCommand,
		DstCommand: c.SrcCommand,
	}
}
