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

package sniffer

import (
	"net"
	"reflect"
	"testing"

	"github.com/google/gopacket/pcap"
)

var formatDeviceNamesTests = []struct {
	name       string
	interfaces []pcap.Interface
	withDesc   bool
	withIP     bool
	want       []string
}{
	{name: "empty"},
	{
		name: "loopback no withs",
		interfaces: []pcap.Interface{
			{
				Name: "lo", Description: "loopback",
				Addresses: []pcap.InterfaceAddress{
					{IP: net.IP{127, 0, 0, 1}, Netmask: net.IPMask{255, 0, 0, 0}},
				},
			},
		},
		want: []string{
			"lo",
		},
	},
	{
		name: "loopback with desc",
		interfaces: []pcap.Interface{
			{
				Name: "lo", Description: "loopback",
				Addresses: []pcap.InterfaceAddress{
					{IP: net.IP{127, 0, 0, 1}, Netmask: net.IPMask{255, 0, 0, 0}},
				},
			},
		},
		withDesc: true,
		want: []string{
			"lo (loopback)",
		},
	},
	{
		name: "loopback with IPs",
		interfaces: []pcap.Interface{
			{
				Name: "lo", Description: "loopback",
				Addresses: []pcap.InterfaceAddress{
					{IP: net.IP{127, 0, 0, 1}, Netmask: net.IPMask{255, 0, 0, 0}},
				},
			},
		},
		withIP: true,
		want: []string{
			"lo (127.0.0.1)",
		},
	},
	{
		name: "loopback with the lot",
		interfaces: []pcap.Interface{
			{
				Name: "lo", Description: "loopback",
				Addresses: []pcap.InterfaceAddress{
					{IP: net.IP{127, 0, 0, 1}, Netmask: net.IPMask{255, 0, 0, 0}},
				},
			},
		},
		withDesc: true,
		withIP:   true,
		want: []string{
			"lo (loopback) (127.0.0.1)",
		},
	},
	{
		name: "two addr loopback with the lot",
		interfaces: []pcap.Interface{
			{
				Name: "lo", Description: "loopback",
				Addresses: []pcap.InterfaceAddress{
					{IP: net.IP{127, 0, 0, 1}, Netmask: net.IPMask{255, 0, 0, 0}},
					{IP: net.IP{127, 0, 1, 1}, Netmask: net.IPMask{255, 0, 0, 0}},
				},
			},
		},
		withDesc: true,
		withIP:   true,
		want: []string{
			"lo (loopback) (127.0.0.1 127.0.1.1)",
		},
	},
	{
		name: "no IP loopback with the lot",
		interfaces: []pcap.Interface{
			{
				Name: "lo", Description: "loopback",
			},
		},
		withDesc: true,
		withIP:   true,
		want: []string{
			"lo (loopback) (Not assigned ip address)",
		},
	},
}

func TestFormatDevices(t *testing.T) {
	for _, test := range formatDeviceNamesTests {
		got := formatDeviceNames(test.interfaces, test.withDesc, test.withIP)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("unexpected result for test %s:\ngot: %v\nwant:%v",
				test.name, got, test.want)
		}
	}
}
