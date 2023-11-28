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
	Incoming PortsForProtocol
	Outgoing PortsForProtocol
}

// ContainsMetrics returns true if the metrics have non-zero data
func (pd PacketData) ContainsMetrics() bool {
	return pd.Incoming.TCP > 0 || pd.Incoming.UDP > 0 || pd.Outgoing.TCP > 0 || pd.Outgoing.UDP > 0
}

// PortsForProtocol tracks counters for TCP/UDP connections
type PortsForProtocol struct {
	TCP uint64
	UDP uint64
}

// CounterUpdateEvent is sent every time we get new packet data for a PID
type CounterUpdateEvent struct {
	pktLen        int
	TransProtocol applayer.Transport
	Proc          *common.ProcessTuple
}

// RequestCounters is a request for packet data
type RequestCounters struct {
	Pid  int
	Resp chan PacketData
}

// Tracker tracks network packets and maps them to a PID
type Tracker struct {
	procData    map[int]PacketData
	dataMut     sync.RWMutex
	procWatcher *procs.ProcessesWatcher

	log    *logp.Logger
	gctime time.Duration

	updateChan chan CounterUpdateEvent
	reqChan    chan RequestCounters
	stopChan   chan struct{}

	// special test helpers
	loopWaiter chan struct{}
	testmode   bool
	// used for the garbage collection subprocess, wrapped for aid of testing
	gcPIDFetch func(ctx context.Context, pid int32) (bool, error)
}
