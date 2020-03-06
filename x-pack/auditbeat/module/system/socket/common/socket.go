// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package common

import (
	"sync"
)

type Socket struct {
	Process *Process

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
	if f.HasKey() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.flows[f.Key()] = f
	}
}

func (s *Socket) Enrich(f *Flow) {
	// if the sock is not bound to a local address yet, update if possible
	if !s.bound && f.Local() != nil {
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

func (s *Socket) SetPID(pid uint32) *Socket {
	if pid != 0 {
		s.pid = pid
	}
	return s
}

func (s *Socket) SetProcess(p *Process) *Socket {
	if p != nil {
		s.Process = p
	}
	return s
}

func (s *Socket) Key() uintptr {
	return s.socket
}
