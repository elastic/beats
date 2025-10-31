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

package flows

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type flowsChan struct {
	ch chan []beat.Event
}

func (f *flowsChan) PublishFlows(events []beat.Event) {
	f.ch <- events
}

func TestFlowsCounting(t *testing.T) {
	logp.TestingSetup()

	mac1 := []byte{1, 2, 3, 4, 5, 6}
	mac2 := []byte{6, 5, 4, 3, 2, 1}
	ip1 := []byte{127, 0, 0, 1}
	ip2 := []byte{128, 0, 1, 2}
	port1 := []byte{0, 1}
	port2 := []byte{0, 2}

	module, err := NewFlows(nil, &procs.ProcessesWatcher{}, &config.Flows{})
	assert.NoError(t, err)

	uint1, err := module.NewUint("uint1")
	assert.NoError(t, err)
	uint2, err := module.NewUint("uint2")
	assert.NoError(t, err)
	int1, err := module.NewInt("int1")
	assert.NoError(t, err)
	int2, err := module.NewInt("int2")
	assert.NoError(t, err)
	float1, err := module.NewFloat("float1")
	assert.NoError(t, err)
	float2, err := module.NewFloat("float2")
	assert.NoError(t, err)

	pub := &flowsChan{make(chan []beat.Event, 1)}

	processor := &flowsProcessor{
		table:    module.table,
		watcher:  &procs.ProcessesWatcher{},
		counters: module.counterReg,
		timeout:  20 * time.Millisecond,
	}
	processor.spool.init(pub.PublishFlows, 1)

	worker, err := makeWorker(
		processor,
		10*time.Millisecond,
		1,
		-1,
		0)
	if err != nil {
		t.Fatalf("Failed to create flow worker: %v", err)
	}

	worker.start()
	defer worker.stop()

	idForward := newFlowID()
	addrForward := addAll(
		addEther(mac1, mac2),
		addIP(ip1, ip2),
		addTCP(port1, port2),
	)
	addrForward(idForward)

	idRev := newFlowID()
	addrRev := addAll(
		addEther(mac2, mac1),
		addIP(ip2, ip1),
		addTCP(port2, port1),
	)
	addrRev(idRev)
	assert.True(t, FlowIDsEqual(idForward, idRev))

	{
		module.Lock()

		flow := module.Get(idForward)
		flowRev := module.Get(idRev)

		int1.Add(flow, -1)
		uint1.Add(flow, 1)
		float1.Add(flow, 3.14)

		int2.Set(flowRev, -1)
		uint2.Set(flowRev, 5)
		float2.Set(flowRev, 1.4142)

		module.Unlock()
	}

	var events []beat.Event
	select {
	case events = <-pub.ch:
	case <-time.After(5 * time.Second):
	}

	if events == nil {
		t.Fatalf("no event received in time")
	}
	event := events[0].Fields
	t.Logf("event: %v", event)

	source := event["source"].(mapstr.M)
	dest := event["destination"].(mapstr.M)
	network := event["network"].(mapstr.M)

	// validate generated event
	assert.Equal(t, formatHardwareAddr(net.HardwareAddr(mac1)), source["mac"])
	assert.Equal(t, formatHardwareAddr(net.HardwareAddr(mac2)), dest["mac"])
	assert.Equal(t, net.IP(ip1).String(), source["ip"])
	assert.Equal(t, net.IP(ip2).String(), dest["ip"])
	assert.Equal(t, uint16(256), source["port"])
	assert.Equal(t, uint16(512), dest["port"])
	assert.Equal(t, "tcp", network["transport"])

	stat := source
	assert.Equal(t, int64(-1), stat["int1"])
	assert.Equal(t, nil, stat["int2"])
	assert.Equal(t, uint64(1), stat["uint1"])
	assert.Equal(t, nil, stat["uint2"])
	assert.Equal(t, 3.14, stat["float1"])
	assert.Equal(t, nil, stat["float2"])

	stat = dest
	assert.Equal(t, nil, stat["int1"])
	assert.Equal(t, int64(-1), stat["int2"])
	assert.Equal(t, nil, stat["uint1"])
	assert.Equal(t, uint64(5), stat["uint2"])
	assert.Equal(t, nil, stat["float1"])
	assert.Equal(t, 1.4142, stat["float2"])
}
