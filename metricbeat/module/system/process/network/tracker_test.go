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

package network

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
)

type testCase struct {
	name     string
	inputs   []counterUpdateEvent
	expected map[int]PacketData
}

func TestPacketGetUpdate(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("network features are linux-only")
	}
	testTrack := &Tracker{
		procData:   make(map[int]PacketData),
		updateChan: make(chan counterUpdateEvent, 10),
		reqChan:    make(chan requestCounters),
		stopChan:   make(chan struct{}),
		testmode:   true,
		gctime:     time.Minute * 10,
	}

	err := testTrack.Track()
	require.NoError(t, err)

	testTrack.Update(40, applayer.TransportTCP, &common.ProcessTuple{Src: common.Process{PID: 11}})

	testTrack.Update(44, applayer.TransportUDP, &common.ProcessTuple{Src: common.Process{PID: 13}})

	require.Eventually(t, func() bool { return testTrack.Get(13).Outgoing.UDP > 0 }, time.Second*10, time.Millisecond)

}

func TestGarbageCollect(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("network features are linux-only")
	}
	_ = logp.DevelopmentSetup()
	testTrack := &Tracker{
		procData:   make(map[int]PacketData),
		updateChan: make(chan counterUpdateEvent, 10),
		reqChan:    make(chan requestCounters),
		stopChan:   make(chan struct{}, 1),
		testmode:   true,
		gctime:     time.Millisecond,
		dataMut:    sync.RWMutex{},
		loopWaiter: make(chan struct{}),
		log:        logp.L(),
	}

	testTrack.gcPIDFetch = func(ctx context.Context, pid int32) (bool, error) {
		if pid == 10 || pid == 1245 {
			return true, nil
		}
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	//start with all pids filled out
	testTrack.procData = map[int]PacketData{10: {}, 1245: {}}

	// start garbage collector
	go func() {
		testTrack.garbageCollect(ctx)
	}()

	<-testTrack.loopWaiter
	testTrack.dataMut.Lock()
	require.Equal(t, map[int]PacketData{10: {}, 1245: {}}, testTrack.procData)
	// remove a pid, test again
	testTrack.gcPIDFetch = func(ctx context.Context, pid int32) (bool, error) {
		if pid == 10 {
			return true, nil
		}
		return false, nil
	}
	testTrack.dataMut.Unlock()
	<-testTrack.loopWaiter

	testTrack.dataMut.Lock()
	require.Equal(t, map[int]PacketData{10: {}}, testTrack.procData)
	testTrack.dataMut.Unlock()
	// gently shut down
	testTrack.Stop()
	<-testTrack.loopWaiter

}

func TestPacketUpdates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("network features are linux-only")
	}
	cases := []testCase{
		{
			name: "base-case",
			inputs: []counterUpdateEvent{
				{
					pktLen:        40,
					TransProtocol: applayer.TransportTCP,
					Proc:          &common.ProcessTuple{Src: common.Process{PID: 11}},
				},
			},
			expected: map[int]PacketData{
				11: {Outgoing: ProtocolCounters{TCP: 40}},
			},
		},
		{
			name: "multiple-proto",
			inputs: []counterUpdateEvent{
				{
					pktLen:        40,
					TransProtocol: applayer.TransportTCP,
					Proc:          &common.ProcessTuple{Src: common.Process{PID: 11}},
				},
				{
					pktLen:        44,
					TransProtocol: applayer.TransportUDP,
					Proc:          &common.ProcessTuple{Src: common.Process{PID: 13}},
				},
				{
					pktLen:        10,
					TransProtocol: applayer.TransportTCP,
					Proc:          &common.ProcessTuple{Src: common.Process{PID: 23}},
				},
				{
					pktLen:        70,
					TransProtocol: applayer.TransportTCP,
					Proc:          &common.ProcessTuple{Dst: common.Process{PID: 11}},
				},
				{
					pktLen:        41,
					TransProtocol: applayer.TransportTCP,
					Proc:          &common.ProcessTuple{Src: common.Process{PID: 11}},
				},
			},
			expected: map[int]PacketData{
				11: {Outgoing: ProtocolCounters{TCP: 81}, Incoming: ProtocolCounters{TCP: 70}},
				13: {Outgoing: ProtocolCounters{UDP: 44}},
				23: {Outgoing: ProtocolCounters{TCP: 10}},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testTrack := &Tracker{
				procData:   make(map[int]PacketData),
				updateChan: make(chan counterUpdateEvent),
				reqChan:    make(chan requestCounters, 10),
				stopChan:   make(chan struct{}),
				testmode:   true,
				gctime:     time.Minute,
			}

			err := testTrack.Track()
			require.NoError(t, err)

			for _, input := range testCase.inputs {

				testTrack.Update(input.pktLen, input.TransProtocol, input.Proc)

			}

			testTrack.Stop()
			require.Equal(t, testCase.expected, testTrack.procData)
		})
	}
}
