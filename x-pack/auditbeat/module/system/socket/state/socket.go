// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"sync"
)

type Socket struct {
	process *Process

	socket uintptr

	mutex sync.RWMutex
	flows map[string]*Flow

	// Sockets have direction if they have been connect()ed or accept()ed.
	direction FlowDirection
	bound     bool
	pid       uint32
}

func CreateSocket(s uintptr, pid uint32) *Socket {
	return &Socket{
		socket: s,
		flows:  make(map[string]*Flow, 1),
	}
}

func (s *Socket) AddFlow(f *Flow) {
	if f.hasKey() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.flows[f.key()] = f
	}
}

func (s *Socket) Enrich(f *Flow) {
	// if the sock is not bound to a local address yet, update if possible
	if !s.bound && f.LocalIP() != nil {
		s.bound = true
		s.mutex.Lock()
		for _, flow := range s.flows {
			flow.local.addr = f.local.addr
		}
		s.mutex.Unlock()
	}

	if s.direction == DirectionUnknown {
		s.direction = f.direction
	}
	f.direction = s.direction

	if s.pid == 0 {
		s.pid = f.pid
	}
	f.pid = s.pid

	f.Socket = s
}

func (s *Socket) GetFlow(key string) *Flow {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.flows[key]
}

func (s *Socket) RemoveFlow(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.flows, key)
}

// returns a shallow copy of the flows
func (s *Socket) Flows() []*Flow {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	flows := make([]*Flow, len(s.flows))
	count := 0
	for _, flow := range s.flows {
		flows[count] = flow
		count++
	}
	return flows
}

func (s *Socket) Key() uintptr {
	return s.socket
}

func (s *Socket) ProcessKey() uint32 {
	return s.pid
}

func (s *Socket) SetProcess(p *Process) {
	if p != nil {
		s.process = p
	}
}
