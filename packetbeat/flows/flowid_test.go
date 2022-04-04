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

//go:build !integration
// +build !integration

package flows

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

type applyAddr func(f *FlowID)

func addEther(a, b []byte) applyAddr {
	return func(f *FlowID) {
		f.AddEth(net.HardwareAddr(a), net.HardwareAddr(b))
	}
}

func addIP(a, b []byte) applyAddr {
	return func(f *FlowID) {
		if len(a) <= 4 {
			f.AddIPv4(net.IP(a), net.IP(b))
		} else {
			f.AddIPv6(net.IP(a), net.IP(b))
		}
	}
}

func addVLan(u []byte) applyAddr {
	id := binary.LittleEndian.Uint16(u)
	return func(f *FlowID) {
		f.AddVLan(id)
	}
}

func addTCP(a, b []byte) applyAddr {
	src := binary.LittleEndian.Uint16(a)
	dst := binary.LittleEndian.Uint16(b)
	return func(f *FlowID) {
		f.AddTCP(src, dst)
	}
}

func addAll(addr ...applyAddr) applyAddr {
	return func(f *FlowID) {
		for _, a := range addr {
			a(f)
		}
	}
}

func vlanAddr(id *FlowID) ([]byte, []byte, bool) {
	v := id.VLan()
	return v, nil, len(v) == 2
}

func outerVlanAddr(id *FlowID) ([]byte, []byte, bool) {
	v := id.OutterVLan()
	return v, nil, len(v) == 2
}

func concat(xs ...[]byte) []byte {
	return bytes.Join(xs, []byte{})
}

func TestFlowIDAddressSorted(t *testing.T) {
	mac1 := []byte{1, 2, 3, 4, 5, 6}
	mac2 := []byte{6, 5, 4, 3, 2, 1}
	ip1 := []byte{127, 0, 0, 1}
	ip2 := []byte{128, 0, 1, 2}
	ip3 := []byte{128, 1, 1, 3}
	ip4 := []byte{129, 2, 1, 4}
	port1 := []byte{0, 1}
	port2 := []byte{0, 2}
	vlan1 := []byte{1, 1}
	vlan2 := []byte{1, 2}
	vlan3 := []byte{1, 3}

	type addrCheck struct {
		getter func(*FlowID) ([]byte, []byte, bool)
		a, b   []byte
	}

	tests := []struct {
		add   applyAddr
		flags []FlowIDFlag
		id    []byte
		addr  []addrCheck
	}{
		{
			addEther(mac1, mac2),
			[]FlowIDFlag{EthFlow},
			concat(mac1, mac2),
			[]addrCheck{
				{(*FlowID).EthAddr, mac1, mac2},
			},
		},
		{
			addEther(mac2, mac1),
			[]FlowIDFlag{EthFlow},
			concat(mac1, mac2),
			[]addrCheck{
				{(*FlowID).EthAddr, mac2, mac1},
			},
		},
		{
			addAll(addEther(mac1, mac2), addEther(mac2, mac1)),
			[]FlowIDFlag{EthFlow},
			concat(mac2, mac1),
			[]addrCheck{
				{(*FlowID).EthAddr, mac2, mac1},
			},
		},
		{
			addIP(ip1, ip2),
			[]FlowIDFlag{IPv4Flow},
			concat(ip1, ip2),
			[]addrCheck{
				{(*FlowID).IPv4Addr, ip1, ip2},
			},
		},
		{
			addIP(ip2, ip1),
			[]FlowIDFlag{IPv4Flow},
			concat(ip1, ip2),
			[]addrCheck{
				{(*FlowID).IPv4Addr, ip2, ip1},
			},
		},
		{
			addAll(addIP(ip2, ip1), addIP(ip3, ip4)),
			[]FlowIDFlag{IPv4Flow},
			concat(ip1, ip2, ip4, ip3),
			[]addrCheck{
				{(*FlowID).OutterIPv4Addr, ip2, ip1},
				{(*FlowID).IPv4Addr, ip3, ip4},
			},
		},
		{
			addTCP(port1, port2),
			[]FlowIDFlag{TCPFlow},
			concat(port1, port2),
			[]addrCheck{
				{(*FlowID).TCPAddr, port1, port2},
			},
		},
		{
			addTCP(port2, port1),
			[]FlowIDFlag{TCPFlow},
			concat(port1, port2),
			[]addrCheck{
				{(*FlowID).TCPAddr, port2, port1},
			},
		},
		{
			addAll(addEther(mac1, mac2), addIP(ip1, ip2)),
			[]FlowIDFlag{EthFlow, IPv4Flow},
			concat(mac1, mac2, ip1, ip2),
			[]addrCheck{
				{(*FlowID).EthAddr, mac1, mac2},
				{(*FlowID).IPv4Addr, ip1, ip2},
			},
		},
		{
			addAll(addEther(mac1, mac2), addIP(ip2, ip1)),
			[]FlowIDFlag{EthFlow, IPv4Flow},
			concat(mac1, mac2, ip2, ip1),
			[]addrCheck{
				{(*FlowID).EthAddr, mac1, mac2},
				{(*FlowID).IPv4Addr, ip2, ip1},
			},
		},
		{
			addAll(addEther(mac2, mac1), addIP(ip1, ip2)),
			[]FlowIDFlag{EthFlow, IPv4Flow},
			concat(mac1, mac2, ip2, ip1),
			[]addrCheck{
				{(*FlowID).EthAddr, mac2, mac1},
				{(*FlowID).IPv4Addr, ip1, ip2},
			},
		},
		{
			addAll(addEther(mac2, mac1), addIP(ip2, ip1)),
			[]FlowIDFlag{EthFlow, IPv4Flow},
			concat(mac1, mac2, ip1, ip2),
			[]addrCheck{
				{(*FlowID).EthAddr, mac2, mac1},
				{(*FlowID).IPv4Addr, ip2, ip1},
			},
		},
		{
			addAll(addEther(mac1, mac2), addVLan(vlan1)),
			[]FlowIDFlag{EthFlow, VLanFlow},
			concat(mac1, mac2, vlan1),
			[]addrCheck{
				{(*FlowID).EthAddr, mac1, mac2},
				{vlanAddr, vlan1, nil},
			},
		},
		{
			addAll(addEther(mac1, mac2), addVLan(vlan1), addVLan(vlan2)),
			[]FlowIDFlag{EthFlow, VLanFlow, OutterVlanFlow},
			concat(mac1, mac2, vlan1, vlan2),
			[]addrCheck{
				{(*FlowID).EthAddr, mac1, mac2},
				{outerVlanAddr, vlan1, nil},
				{vlanAddr, vlan2, nil},
			},
		},
		{
			addAll(addEther(mac1, mac2), addVLan(vlan1), addVLan(vlan2), addVLan(vlan3)),
			[]FlowIDFlag{EthFlow, VLanFlow, OutterVlanFlow},
			concat(mac1, mac2, vlan3, vlan2),
			[]addrCheck{
				{(*FlowID).EthAddr, mac1, mac2},
				{outerVlanAddr, vlan2, nil},
				{vlanAddr, vlan3, nil},
			},
		},
	}

	for i, test := range tests {
		t.Logf("flow id address sorted(%v): %v", i, test)

		id := newFlowID()

		test.add(id)

		for _, flag := range test.flags {
			assert.True(t, (id.Flags()&flag) != 0)
		}

		assert.Equal(t, test.id, id.flowID)

		for _, check := range test.addr {
			a, b, ok := check.getter(id)
			if !ok {
				t.Error("failed to load address from id")
				continue
			}

			assert.Equal(t, check.a, a)
			assert.Equal(t, check.b, b)
		}
	}
}

func TestSimilarWithOffsets(t *testing.T) {
	mac1 := []byte{1, 2, 3, 4, 5, 6}
	mac2 := []byte{6, 5, 4, 3, 2, 1}
	ip1 := []byte{127, 0, 0, 1}
	ip2 := []byte{128, 0, 1, 2}
	ip3 := []byte{127, 0, 0, 1}
	ip4 := []byte{128, 0, 1, 2}

	addr1 := addAll(
		addEther(mac1, mac2),
		addIP(ip1, ip2), addIP(ip3, ip4))
	addr2 := addAll(
		addEther(mac1, mac2),
		addIP(ip2, ip1), addIP(ip3, ip4), addIP(ip1, ip2))

	id1 := newFlowID()
	id2 := newFlowID()
	addr1(id1)
	addr2(id2)

	assert.Equal(t, id1.flowID, id2.flowID)
	assert.Equal(t, id1.flags, id2.flags)
	assert.NotEqual(t, id1.flowIDMeta, id2.flowIDMeta)
}
