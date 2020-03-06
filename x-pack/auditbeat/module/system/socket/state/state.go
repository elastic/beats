// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	socket_common "github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
)

const (
	// how often to collect and report expired and terminated flows.
	reapInterval = time.Second
	// how often the state log generated (only in debug mode).
	logInterval = time.Second * 30
)

type State struct {
	sync.Mutex

	reporter mb.PushReporterV2
	log      helper.Logger

	flows     *flowCache
	sockets   *socketCache
	processes *processCache
	threads   *common.Cache

	done []*socket_common.Flow

	clock *clock
	dns   *dnsTracker

	eventCount uint64
	lastEvents uint64
	lastTime   time.Time
}

func NewState(r mb.PushReporterV2, log helper.Logger, processTimeout, inactiveTimeout, closeTimeout, clockMaxDrift time.Duration) *State {
	s := makeState(r, log, processTimeout, inactiveTimeout*2, inactiveTimeout, closeTimeout, clockMaxDrift)
	go s.reapLoop()
	return s
}

func makeState(r mb.PushReporterV2, log helper.Logger, processTimeout, socketTimeout, inactiveTimeout, closeTimeout, clockMaxDrift time.Duration) *State {
	clock := newClock(log, clockMaxDrift)

	newState := &State{
		reporter: r,
		log:      log,
		clock:    clock,
		dns:      newDNSTracker(socketTimeout),
		done:     []*socket_common.Flow{},
	}
	flowCache := newFlowCache(inactiveTimeout, newState.finalizeFlow)
	socketCache := newSocketCache(socketTimeout, closeTimeout, newState.finalizeSocket)
	processCache := newProcessCache(processTimeout)

	newState.flows = flowCache
	newState.sockets = socketCache
	newState.processes = processCache
	newState.threads = common.NewCache(processTimeout, 8)

	return newState
}

// Flow stuff

// UpdateFlow receives a partial flow and creates or updates an existing flow.
func (s *State) UpdateFlow(f *socket_common.Flow) {
	s.UpdateFlowWithCondition(f, nil)
}

// UpdateFlowWithCondition receives a partial flow and creates or updates an
// existing flow. The optional condition must be met before an existing flow is
// updated. Otherwise the update is ignored.
func (s *State) UpdateFlowWithCondition(f *socket_common.Flow, condition func(*socket_common.Flow) bool) {
	s.Lock()
	defer s.Unlock()

	f.FormatCreated(s.clock.kernelToTime).FormatLastSeen(s.clock.kernelToTime)

	// Get or create a socket for this flow
	socket := s.getOrCreateSocket(f.Ptr(), f.PID)
	if socket != nil {
		// enrich the socket with information from the flow and link up the relationships
		socket.Enrich(f)
		// link up the process with the newly enriched socket
		socket.SetProcess(s.processes.Get(f.PID))
	}

	cached := s.flows.PutIfAbsent(f)

	if cached == nil {
		return
	}

	if cached != f {
		if condition != nil && !condition(cached) {
			return
		}
		cached.Merge(f)
	} else {
		cached.MarkNew()
	}

	s.enrichDNS(cached)
	socket.AddFlow(cached)
}

func (s *State) finalizeFlow(f *socket_common.Flow) {
	f.Terminate()
	if f.IsValid() {
		s.done = append(s.done, f)
	}
}

func (s *State) PopFlows() []*socket_common.Flow {
	r := s.done
	s.done = []*socket_common.Flow{}
	return r
}

// Socket stuff

func (s *State) getOrCreateSocket(socket uintptr, pid uint32) *socket_common.Socket {
	if cached := s.sockets.PutIfAbsent(socket_common.CreateSocket(socket, pid)); cached != nil {
		return cached.SetProcess(s.processes.Get(pid))
	}

	return nil
}

func (s *State) SocketEnd(socket uintptr, pid uint32) {
	s.Lock()
	defer s.Unlock()

	if closed := s.sockets.Close(socket); closed != nil {
		closed.SetPID(pid).SetProcess(s.processes.Get(pid))
	}
}

func (s *State) finalizeSocket(socket *socket_common.Socket) {
	for _, flow := range socket.Flows() {
		s.flows.Evict(flow)
	}
}

// process stuff

func (s *State) ProcessStart(p *socket_common.Process) {
	s.Lock()
	defer s.Unlock()

	s.processes.Put(p.FormatCreatedIfZero(s.clock.kernelToTime))
}

func (s *State) ProcessEnd(pid uint32) {
	s.Lock()
	defer s.Unlock()

	s.processes.Delete(pid)
}

func (s *State) logState() {
	s.Lock()
	done := len(s.done)
	flows := s.flows.Size()
	activeSockets := s.sockets.Size()
	closingSockets := s.sockets.ClosingSize()
	processes := s.processes.Size()
	threads := s.threads.Size()

	events := atomic.LoadUint64(&s.eventCount)

	now := time.Now()
	eventsPerSecond := float64(events-s.lastEvents) * float64(time.Second) / float64(now.Sub(s.lastTime))

	s.lastEvents = events
	s.lastTime = now
	s.Unlock()

	s.log.Debugf("state flows=%d sockets=%d procs=%d threads=%d done=%d closing=%d eps=%.1f", flows, activeSockets, processes, threads, done, closingSockets, eventsPerSecond)
}

func (s *State) reapLoop() {
	reportTicker := time.NewTicker(reapInterval)
	defer reportTicker.Stop()
	logTicker := time.NewTicker(logInterval)
	defer logTicker.Stop()
	for {
		select {
		case <-s.reporter.Done():
			return
		case <-reportTicker.C:
			s.Lock()
			s.CleanUp()
			for _, flow := range s.PopFlows() {
				if flow.IsCurrentProcess() {
					// Do not report flows for which we are the source
					// to prevent a feedback loop.
					continue
				}
				if !s.reporter.Event(flow.ToEvent(true)) {
					s.Unlock()
					return
				}
			}
			s.Unlock()
		case <-logTicker.C:
			s.logState()
		}
	}
}

func (s *State) CleanUp() {
	s.flows.CleanUp()
	s.sockets.CleanUp()
	s.processes.CleanUp()
	s.threads.CleanUp()
	s.dns.cleanUp()
}

func (s *State) PushThreadEvent(thread uint32, e socket_common.Event) {
	s.Lock()
	defer s.Unlock()

	s.threads.PutIfAbsent(thread, e)
}

func (s *State) PopThreadEvent(thread uint32) socket_common.Event {
	s.Lock()
	defer s.Unlock()

	value := s.threads.Delete(thread)
	if value == nil {
		return nil
	}
	return value.(socket_common.Event)
}

func (s *State) SyncClocks(kernelNanos, userNanos uint64) {
	s.Lock()
	defer s.Unlock()

	s.clock.sync(kernelNanos, userNanos)
}

func (s *State) OnDNSTransaction(tr dns.Transaction) error {
	s.Lock()
	defer s.Unlock()

	s.dns.addTransaction(tr)
	return nil
}

func (s *State) Increment() {
	atomic.AddUint64(&s.eventCount, 1)
}

func (s *State) enrichDNS(f *socket_common.Flow) {
	if address := f.DNSAddress(); address != nil {
		s.dns.registerEndpoint(address, f.Process)
	}
}
