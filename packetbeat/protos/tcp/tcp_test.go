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

package tcp

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/stretchr/testify/assert"
)

// Test Constants
const (
	ServerIP   = "192.168.0.1"
	ServerPort = 12345
	ClientIP   = "10.0.0.1"
)

var httpProtocol, mysqlProtocol, redisProtocol protos.Protocol

func init() {
	new := func(_ bool, _ protos.Reporter, _ procs.ProcessesWatcher, _ *conf.C) (protos.Plugin, error) {
		return &TestProtocol{}, nil
	}

	protos.Register("httpTest", new)
	protos.Register("mysqlTest", new)
	protos.Register("redisTest", new)

	httpProtocol = protos.Lookup("httpTest")
	redisProtocol = protos.Lookup("redisTest")
	mysqlProtocol = protos.Lookup("mysqlTest")
}

type TestProtocol struct {
	Ports []int

	init  func(testMode bool, results protos.Reporter) error
	parse func(*protos.Packet, *common.TCPTuple, uint8, protos.ProtocolData) protos.ProtocolData
	onFin func(*common.TCPTuple, uint8, protos.ProtocolData) protos.ProtocolData
	gap   func(*common.TCPTuple, uint8, int, protos.ProtocolData) (protos.ProtocolData, bool)
}

var _ protos.Plugin = &TestProtocol{
	init: func(m bool, r protos.Reporter) error { return nil },
	parse: func(p *protos.Packet, t *common.TCPTuple, d uint8, priv protos.ProtocolData) protos.ProtocolData {
		return priv
	},
	onFin: func(t *common.TCPTuple, d uint8, p protos.ProtocolData) protos.ProtocolData {
		return p
	},
	gap: func(t *common.TCPTuple, d uint8, b int, p protos.ProtocolData) (protos.ProtocolData, bool) {
		return p, true
	},
}

func (proto *TestProtocol) Init(testMode bool, results protos.Reporter) error {
	return proto.init(testMode, results)
}

func (proto TestProtocol) GetPorts() []int {
	return proto.Ports
}

func (proto TestProtocol) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {
	return proto.parse(pkt, tcptuple, dir, private)
}

func (proto TestProtocol) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return proto.onFin(tcptuple, dir, private)
}

func (proto TestProtocol) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	return proto.gap(tcptuple, dir, nbytes, private)
}

func (proto TestProtocol) ConnectionTimeout() time.Duration {
	return 0
}

func Test_configToPortsMap(t *testing.T) {
	type configTest struct {
		Input  map[protos.Protocol]protos.TCPPlugin
		Output map[uint16]protos.Protocol
	}

	configTests := []configTest{
		{
			Input: map[protos.Protocol]protos.TCPPlugin{
				httpProtocol: &TestProtocol{Ports: []int{80, 8080}},
			},
			Output: map[uint16]protos.Protocol{
				80:   httpProtocol,
				8080: httpProtocol,
			},
		},
		{
			Input: map[protos.Protocol]protos.TCPPlugin{
				httpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				mysqlProtocol: &TestProtocol{Ports: []int{3306}},
				redisProtocol: &TestProtocol{Ports: []int{6379, 6380}},
			},
			Output: map[uint16]protos.Protocol{
				80:   httpProtocol,
				8080: httpProtocol,
				3306: mysqlProtocol,
				6379: redisProtocol,
				6380: redisProtocol,
			},
		},

		// should ignore duplicate ports in the same protocol
		{
			Input: map[protos.Protocol]protos.TCPPlugin{
				httpProtocol:  &TestProtocol{Ports: []int{80, 8080, 8080}},
				mysqlProtocol: &TestProtocol{Ports: []int{3306}},
			},
			Output: map[uint16]protos.Protocol{
				80:   httpProtocol,
				8080: httpProtocol,
				3306: mysqlProtocol,
			},
		},
	}

	for _, test := range configTests {
		output, err := buildPortsMap(test.Input)
		assert.NoError(t, err)
		assert.Equal(t, test.Output, output)
	}
}

func Test_configToPortsMap_negative(t *testing.T) {
	type errTest struct {
		Input map[protos.Protocol]protos.TCPPlugin
		Err   string
	}

	tests := []errTest{
		{
			// should raise error on duplicate port
			Input: map[protos.Protocol]protos.TCPPlugin{
				httpProtocol:  &TestProtocol{Ports: []int{80, 8080}},
				mysqlProtocol: &TestProtocol{Ports: []int{3306}},
				redisProtocol: &TestProtocol{Ports: []int{6379, 6380, 3306}},
			},
			Err: "Duplicate port (3306) exists",
		},
	}

	for _, test := range tests {
		_, err := buildPortsMap(test.Input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), test.Err)
	}
}

// Mock protos.Protocols used for testing the tcp package.
type protocols struct {
	tcp map[protos.Protocol]protos.TCPPlugin
}

// Verify protocols implements the protos.Protocols interface.
var _ protos.Protocols = &protocols{}

func (p protocols) BpfFilter(withVlans bool, withICMP bool) string       { return "" }
func (p protocols) GetTCP(proto protos.Protocol) protos.TCPPlugin        { return p.tcp[proto] }
func (p protocols) GetUDP(proto protos.Protocol) protos.UDPPlugin        { return nil }
func (p protocols) GetAll() map[protos.Protocol]protos.Plugin            { return nil }
func (p protocols) GetAllTCP() map[protos.Protocol]protos.TCPPlugin      { return p.tcp }
func (p protocols) GetAllUDP() map[protos.Protocol]protos.UDPPlugin      { return nil }
func (p protocols) Register(proto protos.Protocol, plugin protos.Plugin) {}

func TestTCSeqPayload(t *testing.T) {
	type segment struct {
		seq     uint32
		payload []byte
	}

	tests := []struct {
		name          string
		segments      []segment
		expectedGaps  int
		expectedState []byte
	}{
		{
			"No overlap",
			[]segment{
				{1, []byte{1, 2, 3, 4, 5}},
				{6, []byte{6, 7, 8, 9, 10}},
			},
			0,
			[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			"Gap drop state",
			[]segment{
				{1, []byte{1, 2, 3, 4}},
				{15, []byte{5, 6, 7, 8}},
			},
			10,
			[]byte{5, 6, 7, 8},
		},
		{
			"ACK same sequence number",
			[]segment{
				{1, []byte{1, 2}},
				{3, nil},
				{3, []byte{3, 4}},
				{5, []byte{5, 6}},
			},
			0,
			[]byte{1, 2, 3, 4, 5, 6},
		},
		{
			"ACK same sequence number 2",
			[]segment{
				{1, nil},
				{2, nil},
				{2, []byte{1, 2}},
				{4, nil},
				{4, []byte{3, 4}},
				{6, []byte{5, 6}},
				{8, []byte{7, 8}},
				{10, nil},
			},
			0,
			[]byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			"Overlap, first segment bigger",
			[]segment{
				{1, []byte{1, 2}},
				{3, []byte{3, 4}},
				{3, []byte{3}},
				{5, []byte{5, 6}},
			},
			0,
			[]byte{1, 2, 3, 4, 5, 6},
		},
		{
			"Overlap, second segment bigger",
			[]segment{
				{1, []byte{1, 2}},
				{3, []byte{3}},
				{3, []byte{3, 4}},
				{5, []byte{5, 6}},
			},
			0,
			[]byte{1, 2, 3, 4, 5, 6},
		},
		{
			"Overlap, covered",
			[]segment{
				{1, []byte{1, 2, 3, 4}},
				{2, []byte{2, 3}},
				{5, []byte{5, 6}},
			},
			0,
			[]byte{1, 2, 3, 4, 5, 6},
		},
	}

	for i, test := range tests {
		t.Logf("Test (%v): %v", i, test.name)

		gap := 0
		var state []byte
		tcp, err := NewTCP(protocols{
			tcp: map[protos.Protocol]protos.TCPPlugin{
				httpProtocol: &TestProtocol{
					Ports: []int{ServerPort},
					gap:   makeCountGaps(nil, &gap),
					parse: makeCollectPayload(&state, true),
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		addr := common.NewIPPortTuple(4,
			net.ParseIP(ServerIP), ServerPort,
			net.ParseIP(ClientIP), uint16(rand.Intn(65535)))

		for _, segment := range test.segments {
			hdr := &layers.TCP{Seq: segment.seq}
			pkt := &protos.Packet{
				Ts:      time.Now(),
				Tuple:   addr,
				Payload: segment.payload,
			}
			tcp.Process(nil, hdr, pkt)
		}

		assert.Equal(t, test.expectedGaps, gap)
		if len(test.expectedState) != len(state) {
			assert.Equal(t, len(test.expectedState), len(state))
			continue
		}
		assert.Equal(t, test.expectedState, state)
	}
}

// Benchmark that runs with parallelism to help find concurrency related
// issues. To run with parallelism, the 'go test' cpu flag must be set
// greater than 1, otherwise it just runs concurrently but not in parallel.
func BenchmarkParallelProcess(b *testing.B) {
	rand.Seed(18)
	p := protocols{}
	p.tcp = make(map[protos.Protocol]protos.TCPPlugin)
	p.tcp[1] = &TestProtocol{Ports: []int{ServerPort}}
	tcp, _ := NewTCP(p)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pkt := &protos.Packet{
				Ts: time.Now(),
				Tuple: common.NewIPPortTuple(4,
					net.ParseIP(ServerIP), ServerPort,
					net.ParseIP(ClientIP), uint16(rand.Intn(65535))),
				Payload: []byte{1, 2, 3, 4},
			}
			tcp.Process(nil, &layers.TCP{}, pkt)
		}
	})
}

func makeCountGaps(
	counter *int,
	bytes *int,
) func(*common.TCPTuple, uint8, int, protos.ProtocolData) (protos.ProtocolData, bool) {
	return func(
		t *common.TCPTuple,
		d uint8,
		n int,
		p protos.ProtocolData,
	) (protos.ProtocolData, bool) {
		if counter != nil {
			(*counter)++
		}
		if bytes != nil {
			*bytes += n
		}

		return p, true // drop state
	}
}

func makeCollectPayload(
	state *[]byte,
	resetOnNil bool,
) func(*protos.Packet, *common.TCPTuple, uint8, protos.ProtocolData) protos.ProtocolData {
	return func(
		p *protos.Packet,
		t *common.TCPTuple,
		d uint8,
		priv protos.ProtocolData,
	) protos.ProtocolData {
		if resetOnNil && priv == nil {
			(*state) = nil
		}
		*state = append(*state, p.Payload...)
		return *state
	}
}
