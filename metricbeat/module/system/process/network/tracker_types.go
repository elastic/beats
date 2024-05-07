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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
)

// PacketData tracks all counters for a given port
type PacketData struct {
	Incoming ProtocolCounters
	Outgoing ProtocolCounters
}

// ContainsMetrics returns true if the metrics have non-zero data
func (pd PacketData) ContainsMetrics() bool {
	return pd != PacketData{}
}

// ProtocolCounters tracks counters for TCP/UDP connections
type ProtocolCounters struct {
	TCP uint64
	UDP uint64
}

// counterUpdateEvent is sent every time we get new packet data for a PID
type counterUpdateEvent struct {
	pktLen        int
	TransProtocol applayer.Transport
	Proc          *common.ProcessTuple
}

// requestCounters is a request for packet data
type requestCounters struct {
	PID  int
	Resp chan PacketData
}

// Tracker tracks network packets and maps them to a PID
type Tracker struct {
	procData    map[int]PacketData
	dataMut     sync.RWMutex
	procWatcher *procs.ProcessesWatcher

	log    *logp.Logger
	gctime time.Duration

	updateChan chan counterUpdateEvent
	reqChan    chan requestCounters
	stopChan   chan struct{}

	// special test helpers
	loopWaiter chan struct{}
	testmode   bool
	// used for the garbage collection subprocess, wrapped for aid of testing
	gcPIDFetch func(ctx context.Context, pid int32) (bool, error)
}
