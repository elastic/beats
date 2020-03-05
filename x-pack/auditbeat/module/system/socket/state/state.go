// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
)

type State struct {
	sync.Mutex

	reporter mb.PushReporterV2
	log      helper.Logger

	flows     *FlowCache
	sockets   *SocketCache
	processes *ProcessCache
	threads   *common.Cache

	done []*Flow

	clock *clock
	dns   *DNSTracker

	EventCount uint64
}

func NewState(r mb.PushReporterV2, log helper.Logger, processTimeout, inactiveTimeout, closeTimeout, clockMaxDrift time.Duration) *State {
	s := MakeState(r, log, processTimeout, processTimeout, inactiveTimeout, closeTimeout, clockMaxDrift)
	go s.reapLoop()
	return s
}

func MakeState(r mb.PushReporterV2, log helper.Logger, processTimeout, socketTimeout, inactiveTimeout, closeTimeout, clockMaxDrift time.Duration) *State {
	clock := newClock(log, clockMaxDrift)

	newState := &State{
		reporter: r,
		log:      log,
		clock:    clock,
		dns:      NewDNSTracker(processTimeout * 2),
		done:     []*Flow{},
	}
	flowCache := NewFlowCache(inactiveTimeout, newState.finalizeFlow)
	socketCache := NewSocketCache(socketTimeout, closeTimeout, newState.finalizeSocket)
	processCache := NewProcessCache(processTimeout)

	newState.flows = flowCache
	newState.sockets = socketCache
	newState.processes = processCache
	newState.threads = common.NewCache(processTimeout, 8)

	return newState
}

// Flow stuff

// UpdateFlow receives a partial flow and creates or updates an existing flow.
func (s *State) UpdateFlow(f *Flow) {
	s.UpdateFlowWithCondition(f, nil)
}

// UpdateFlowWithCondition receives a partial flow and creates or updates an
// existing flow. The optional condition must be met before an existing flow is
// updated. Otherwise the update is ignored.
func (s *State) UpdateFlowWithCondition(f *Flow, condition func(*Flow) bool) {
	s.Lock()
	defer s.Unlock()

	f.createdTime = s.clock.KernelToTime(f.created)
	f.lastSeenTime = s.clock.KernelToTime(f.lastSeen)

	// Get or create a socket for this flow
	socket := s.getOrCreateSocket(f.socket, f.pid)
	if socket != nil {
		// enrich the socket with information from the flow and link up the relationships
		socket.Enrich(f)
		// link up the process with the newly enriched socket
		if process := s.processes.Get(socket.pid); process != nil {
			socket.Process = process
		}
	}

	cached := s.flows.PutIfAbsent(f)

	if cached == nil {
		return
	}

	if cached != f {
		if condition != nil && !condition(cached) {
			return
		}
		cached.updateWith(f)
	} else {
		cached.createdTime = cached.lastSeenTime
	}

	s.enrichDNS(cached)
	socket.AddFlow(cached)
}

func (s *State) finalizeFlow(f *Flow) {
	f.Terminate()
	if f.IsValid() {
		s.done = append(s.done, f)
	}
}

func (s *State) PopFlows() []*Flow {
	r := s.done
	s.done = []*Flow{}
	return r
}

// Socket stuff

func (s *State) getOrCreateSocket(socket uintptr, pid uint32) *Socket {
	if cached := s.sockets.PutIfAbsent(CreateSocket(socket, pid)); cached != nil {
		if process := s.processes.Get(pid); process != nil {
			cached.Process = process
		}
		return cached
	}

	return nil
}

func (s *State) SocketEnd(socket uintptr, pid uint32) {
	s.Lock()
	defer s.Unlock()

	if closed := s.sockets.Close(socket); closed != nil {
		closed.pid = pid
		if process := s.processes.Get(pid); process != nil {
			closed.Process = process
		}
	}
}

func (s *State) finalizeSocket(socket *Socket) {
	for _, flow := range socket.Flows() {
		s.flows.Evict(flow)
	}
}

// process stuff

func (s *State) ProcessStart(p *Process) {
	s.Lock()
	defer s.Unlock()

	if p.createdTime == (time.Time{}) {
		p.createdTime = s.clock.KernelToTime(p.created)
	}

	s.processes.Put(p)
}

func (s *State) ProcessEnd(pid uint32) {
	s.Lock()
	defer s.Unlock()

	s.processes.Delete(pid)
}

var lastEvents uint64
var lastTime time.Time

func (s *State) logState() {
	s.Lock()
	done := len(s.done)
	flows := s.flows.Size()
	activeSockets := s.sockets.Size()
	closingSockets := s.sockets.ClosingSize()
	processes := s.processes.Size()
	threads := s.threads.Size()
	s.Unlock()

	events := atomic.LoadUint64(&s.EventCount)

	now := time.Now()
	eventsPerSecond := float64(events-lastEvents) * float64(time.Second) / float64(now.Sub(lastTime))

	lastEvents = events
	lastTime = now

	s.log.Debugf("state flows=%d sockets=%d procs=%d threads=%d done=%d closing=%d eps=%.1f", flows, activeSockets, processes, threads, done, closingSockets, eventsPerSecond)
}

func (s *State) reapLoop() {
	pid := os.Getpid()
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
				if int(flow.pid) == pid {
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
	s.dns.CleanUp()
}

func (s *State) PushThreadEvent(thread uint32, e interface{}) {
	s.Lock()
	defer s.Unlock()

	s.threads.PutIfAbsent(thread, e)
}

func (s *State) PopThreadEvent(thread uint32) interface{} {
	s.Lock()
	defer s.Unlock()

	value := s.threads.Delete(thread)
	if value == nil {
		return nil
	}
	return value
}

func (s *State) SyncClocks(kernelNanos, userNanos uint64) {
	s.Lock()
	defer s.Unlock()

	s.clock.Sync(kernelNanos, userNanos)
}

func (s *State) OnDNSTransaction(tr dns.Transaction) error {
	s.Lock()
	defer s.Unlock()

	s.dns.AddTransaction(tr)
	return nil
}

func (s *State) enrichDNS(f *Flow) {
	if f.remote.addr.Port == 53 && f.proto == ProtoUDP && f.pid != 0 && f.Process != nil {
		s.dns.RegisterEndpoint(net.UDPAddr{
			IP:   f.local.addr.IP,
			Port: f.local.addr.Port,
		}, f.Process)
	}
}
