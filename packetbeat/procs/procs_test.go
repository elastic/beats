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

// +build !integration

package procs

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/packetbeat/protos/applayer"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type testingImpl struct {
	localIPs     []net.IP
	portToPID    map[applayer.Transport]map[uint16]int
	pidToCmdline map[int]string
}

type runningProcess struct {
	cmdline string
	pid     int
	ports   []uint16
	proto   applayer.Transport
}

func newTestingImpl(localIPs []net.IP, processes []runningProcess) *testingImpl {
	impl := &testingImpl{
		localIPs: localIPs,
		portToPID: map[applayer.Transport]map[uint16]int{
			applayer.TransportTCP: make(map[uint16]int),
			applayer.TransportUDP: make(map[uint16]int),
		},
		pidToCmdline: make(map[int]string),
	}
	for _, proc := range processes {
		for _, port := range proc.ports {
			impl.portToPID[proc.proto][port] = proc.pid
		}
		impl.pidToCmdline[proc.pid] = proc.cmdline
	}
	return impl
}

func (impl *testingImpl) GetLocalPortToPIDMapping(transport applayer.Transport) (ports map[uint16]int, err error) {
	return impl.portToPID[transport], nil
}

func (impl *testingImpl) GetProcessCommandLine(pid int) string {
	if cmdline, ok := impl.pidToCmdline[pid]; ok {
		return cmdline
	}
	return ""
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
			net.ParseIP("192.168.1.1"),
			net.ParseIP("7777::33"),
		},
		[]runningProcess{
			{
				cmdline: "curl -o /dev/null http://example.net/",
				pid:     101,
				ports:   []uint16{65535},
				proto:   applayer.TransportTCP,
			},
			{
				cmdline: "/usr/X11/bin/webbrowser",
				pid:     102,
				ports:   []uint16{3201, 3202, 3203},
				proto:   applayer.TransportTCP,
			},
			{
				cmdline: "nc -v -l -p 80",
				pid:     105,
				ports:   []uint16{80},
				proto:   applayer.TransportTCP,
			},
			{
				cmdline: "bind",
				pid:     333,
				ports:   []uint16{53},
				proto:   applayer.TransportUDP,
			},
		})
	procs := ProcessesWatcher{}
	err := procs.initWithImpl(config, impl)
	assert.NoError(t, err)

	for idx, testCase := range []struct {
		name                                   string
		srcIP, dstIP, src, dst, srcCmd, dstCmd string
		srcPort, dstPort                       int
		proto                                  applayer.Transport
		preAction                              func()
	}{
		{
			name:  "Unrelated local HTTP client",
			proto: applayer.TransportTCP,
			srcIP: "127.0.0.1", srcPort: 12345,
			dstIP: "1.2.3.4", dstPort: 80,
			src: "", srcCmd: "",
			dst: "", dstCmd: "",
		},
		{
			name:  "Web browser (IPv6)",
			proto: applayer.TransportTCP,
			srcIP: "7777::0:33", srcPort: 3201,
			dstIP: "1234:1234::AAAA", dstPort: 443,
			src: "", srcCmd: "/usr/X11/bin/webbrowser",
			dst: "", dstCmd: "",
		},
		{
			name:  "Curl request",
			proto: applayer.TransportTCP,
			srcIP: "192.168.1.1", srcPort: 65535,
			dstIP: "1.1.1.1", dstPort: 80,
			src: "Curl", srcCmd: "curl -o /dev/null http://example.net/",
			dst: "", dstCmd: "",
		},
		{
			name:  "Unrelated UDP using same port as TCP",
			proto: applayer.TransportUDP,
			srcIP: "192.168.1.1", srcPort: 65535,
			dstIP: "1.1.1.1", dstPort: 80,
			src: "", srcCmd: "",
			dst: "", dstCmd: "",
		},
		{
			name:  "Local web browser to netcat server",
			proto: applayer.TransportTCP,
			srcIP: "127.0.0.1", srcPort: 3202,
			dstIP: "127.0.0.1", dstPort: 80,
			src: "", srcCmd: "/usr/X11/bin/webbrowser",
			dst: "NetCat", dstCmd: "nc -v -l -p 80",
		},
		{
			name:  "External to netcat server",
			proto: applayer.TransportTCP,
			srcIP: "192.168.1.2", srcPort: 3203,
			dstIP: "192.168.1.1", dstPort: 80,
			src: "", srcCmd: "",
			dst: "NetCat", dstCmd: "nc -v -l -p 80",
		},
		{
			name: "New client",
			preAction: func() {
				// add a new running process
				impl.pidToCmdline[555] = "/usr/bin/nmap -sT -P443 10.0.0.0/8"
				impl.portToPID[applayer.TransportTCP][55555] = 555
			},
			proto: applayer.TransportTCP,
			srcIP: "7777::33", srcPort: 55555,
			dstIP: "10.1.2.3", dstPort: 443,
			src: "NMap", srcCmd: "/usr/bin/nmap -sT -P443 10.0.0.0/8",
			dst: "", dstCmd: "",
		},
		{
			name:  "DNS request (UDP)",
			proto: applayer.TransportUDP,
			srcIP: "1234:5678::55", srcPort: 533,
			dstIP: "7777::33", dstPort: 53,
			src: "", srcCmd: "",
			dst: "", dstCmd: "bind",
		},
	} {
		msg := fmt.Sprintf("test case #%d: %s", idx+1, testCase.name)

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
		assert.NotNil(t, result, msg)

		assert.Equal(t, testCase.src, string(result.Src), msg)
		assert.Equal(t, testCase.dst, string(result.Dst), msg)
		assert.Equal(t, testCase.srcCmd, string(result.SrcCommand), msg)
		assert.Equal(t, testCase.dstCmd, string(result.DstCommand), msg)
	}

}
