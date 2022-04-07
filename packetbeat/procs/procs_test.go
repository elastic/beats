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

package procs

import (
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/packetbeat/protos/applayer"
)

type testingImpl struct {
	localIPs     []net.IP
	portToPID    map[applayer.Transport]map[endpoint]int
	pidToProcess map[int]*process
}

type runningProcess struct {
	process
	ports []endpoint
	proto applayer.Transport
}

func newTestingImpl(localIPs []net.IP, processes []runningProcess) *testingImpl {
	impl := &testingImpl{
		localIPs: localIPs,
		portToPID: map[applayer.Transport]map[endpoint]int{
			applayer.TransportTCP: make(map[endpoint]int),
			applayer.TransportUDP: make(map[endpoint]int),
		},
		pidToProcess: make(map[int]*process),
	}
	for i, proc := range processes {
		for _, port := range proc.ports {
			impl.portToPID[proc.proto][port] = proc.pid
		}

		impl.pidToProcess[proc.pid] = &processes[i].process
	}
	return impl
}

func (impl *testingImpl) GetLocalPortToPIDMapping(transport applayer.Transport) (ports map[endpoint]int, err error) {
	return impl.portToPID[transport], nil
}

func (impl *testingImpl) GetProcess(pid int) *process {
	if cmdline, ok := impl.pidToProcess[pid]; ok {
		return cmdline
	}
	return nil
}

func (impl *testingImpl) GetLocalIPs() ([]net.IP, error) {
	return impl.localIPs, nil
}

func TestFindProcessTuple(t *testing.T) {
	logp.TestingSetup()
	config := ProcsConfig{
		Enabled: true,
		Monitored: []ProcConfig{
			{Process: "NetCat", CmdlineGrep: "nc "},
			{Process: "Curl", CmdlineGrep: "curl"},
			{Process: "NMap", CmdlineGrep: "nmap"},
		},
	}
	impl := newTestingImpl(
		[]net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("192.168.1.1"),
			net.ParseIP("7777::33"),
		},
		[]runningProcess{
			{
				process: process{
					name: "mylocal_service",
					args: strings.Fields("/usr/bin/mylocal_service"),
					pid:  9997,
				},
				ports: []endpoint{
					{address: "127.0.0.1", port: 38842},
				},
				proto: applayer.TransportTCP,
			},
			{
				process: process{
					name: "myexternal_service",
					args: strings.Fields("/usr/local/bin/myexternal_service"),
					pid:  9998,
				},
				ports: []endpoint{
					{address: "192.168.1.1", port: 38842},
				},
				proto: applayer.TransportTCP,
			},
			{
				process: process{
					name: "ipv6_only_app",
					args: strings.Fields("/opt/someapp/ipv6_only_app"),
					pid:  9999,
				},
				ports: []endpoint{
					{address: anyIPv6, port: 38842},
				},
				proto: applayer.TransportTCP,
			},
			{
				process: process{
					name: "curl",
					args: strings.Fields("curl -o /dev/null http://example.net/"),
					pid:  101,
				},
				ports: []endpoint{
					{address: anyIPv4, port: 65535},
				},
				proto: applayer.TransportTCP,
			},
			{
				process: process{
					name: "webbrowser",
					args: strings.Fields("/usr/X11/bin/webbrowser"),
					pid:  102,
				},
				ports: []endpoint{
					{anyIPv4, 3201},
					{anyIPv6, 3201},
					{anyIPv4, 3202},
					{anyIPv4, 3203},
				},
				proto: applayer.TransportTCP,
			},
			{
				process: process{
					name: "nc",
					args: strings.Fields("nc -v -l -p 80"),
					pid:  105,
				},
				ports: []endpoint{
					{anyIPv4, 80},
				},
				proto: applayer.TransportTCP,
			},
			{
				process: process{
					name: "bind",
					args: strings.Fields("bind"),
					pid:  333,
				},
				ports: []endpoint{
					{anyIPv6, 53},
				},
				proto: applayer.TransportUDP,
			},
		})
	procs := ProcessesWatcher{}
	err := procs.initWithImpl(config, impl)
	assert.NoError(t, err)

	for _, testCase := range []struct {
		name                   string
		srcIP, dstIP, src, dst string
		srcArgs, dstArgs       []string
		srcPort, dstPort       int
		proto                  applayer.Transport
		preAction              func()
	}{
		{
			name:  "Unrelated local HTTP client",
			proto: applayer.TransportTCP,
			srcIP: "127.0.0.1", srcPort: 12345,
			dstIP: "1.2.3.4", dstPort: 80,
			src: "", srcArgs: nil,
			dst: "", dstArgs: nil,
		},
		{
			name:  "Web browser (IPv6)",
			proto: applayer.TransportTCP,
			srcIP: "7777::0:33", srcPort: 3201,
			dstIP: "1234:1234::AAAA", dstPort: 443,
			src: "webbrowser", srcArgs: strings.Fields("/usr/X11/bin/webbrowser"),
			dst: "", dstArgs: nil,
		},
		{
			name:  "Curl request",
			proto: applayer.TransportTCP,
			srcIP: "192.168.1.1", srcPort: 65535,
			dstIP: "1.1.1.1", dstPort: 80,
			src: "Curl", srcArgs: strings.Fields("curl -o /dev/null http://example.net/"),
			dst: "", dstArgs: nil,
		},
		{
			name:  "Unrelated UDP using same port as TCP",
			proto: applayer.TransportUDP,
			srcIP: "192.168.1.1", srcPort: 65535,
			dstIP: "1.1.1.1", dstPort: 80,
			src: "", srcArgs: nil,
			dst: "", dstArgs: nil,
		},
		{
			name:  "Local web browser to netcat server",
			proto: applayer.TransportTCP,
			srcIP: "127.0.0.1", srcPort: 3202,
			dstIP: "127.0.0.1", dstPort: 80,
			src: "webbrowser", srcArgs: strings.Fields("/usr/X11/bin/webbrowser"),
			dst: "NetCat", dstArgs: strings.Fields("nc -v -l -p 80"),
		},
		{
			name:  "External to netcat server",
			proto: applayer.TransportTCP,
			srcIP: "192.168.1.2", srcPort: 3203,
			dstIP: "192.168.1.1", dstPort: 80,
			src: "", srcArgs: nil,
			dst: "NetCat", dstArgs: strings.Fields("nc -v -l -p 80"),
		},
		{
			name: "New client",
			preAction: func() {
				// add a new running process
				impl.pidToProcess[555] = &process{args: strings.Fields("/usr/bin/nmap -sT -P443 10.0.0.0/8")}
				impl.portToPID[applayer.TransportTCP][endpoint{anyIPv6, 55555}] = 555
			},
			proto: applayer.TransportTCP,
			srcIP: "7777::33", srcPort: 55555,
			dstIP: "10.1.2.3", dstPort: 443,
			src: "NMap", srcArgs: strings.Fields("/usr/bin/nmap -sT -P443 10.0.0.0/8"),
			dst: "", dstArgs: nil,
		},
		{
			name:  "DNS request (UDP)",
			proto: applayer.TransportUDP,
			srcIP: "1234:5678::55", srcPort: 533,
			dstIP: "7777::33", dstPort: 53,
			src: "", srcArgs: nil,
			dst: "bind", dstArgs: strings.Fields("bind"),
		},
		{
			name:  "Local bound port",
			proto: applayer.TransportTCP,
			srcIP: "127.0.0.1", srcPort: 38841,
			dstIP: "127.0.0.1", dstPort: 38842,
			src: "", srcArgs: nil,
			dst: "mylocal_service", dstArgs: strings.Fields("/usr/bin/mylocal_service"),
		},
		{
			name:  "Network bound port",
			proto: applayer.TransportTCP,
			srcIP: "192.168.255.37", srcPort: 65535,
			dstIP: "192.168.1.1", dstPort: 38842,
			src: "", srcArgs: nil,
			dst: "myexternal_service", dstArgs: strings.Fields("/usr/local/bin/myexternal_service"),
		},
		{
			name:  "IPv6 bound port",
			proto: applayer.TransportTCP,
			srcIP: "7fff::11", srcPort: 38842,
			dstIP: "7777::33", dstPort: 38842,
			src: "", srcArgs: nil,
			dst: "ipv6_only_app", dstArgs: strings.Fields("/opt/someapp/ipv6_only_app"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.preAction != nil {
				testCase.preAction()
			}
			input := common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP:   net.ParseIP(testCase.srcIP),
					SrcPort: uint16(testCase.srcPort),
					DstIP:   net.ParseIP(testCase.dstIP),
					DstPort: uint16(testCase.dstPort),
				},
			}
			result := procs.FindProcessesTuple(&input, testCase.proto)
			// nil result is not valid
			assert.NotNil(t, result)

			assert.Equal(t, testCase.src, result.Src.Name)
			assert.Equal(t, testCase.dst, result.Dst.Name)
			assert.Equal(t, testCase.srcArgs, result.Src.Args)
			assert.Equal(t, testCase.dstArgs, result.Dst.Args)
		})
	}
}
