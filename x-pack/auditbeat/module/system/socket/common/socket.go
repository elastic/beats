// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package common

type Socket struct {
	Process *Process

	socket uintptr

	flows map[string]*Flow

	// Sockets have direction if they have been connect()ed or accept()ed.
	direction FlowDirection
	bound     bool
	PID       uint32
}

func CreateSocket(s uintptr, pid uint32) *Socket {
	return &Socket{
		socket: s,
		flows:  make(map[string]*Flow, 1),
	}
}

func (s *Socket) AddFlow(f *Flow) {
	if f.HasKey() {
		s.flows[f.Key()] = f
	}
}

func (s *Socket) Enrich(f *Flow) {
	// if the sock is not bound to a local address yet, update if possible
	if !s.bound && f.Local != nil && f.Local.addr != nil {
		s.bound = true
		for _, flow := range s.flows {
			flow.Local.addr = f.Local.addr
		}
	}

	if s.direction == DirectionUnknown {
		s.direction = f.Direction
	}
	f.Direction = s.direction

	if s.PID == 0 {
		s.PID = f.PID
	}
	f.PID = s.PID

	f.Socket = s
}

func (s *Socket) removeFlow(key string) {
	delete(s.flows, key)
}

// returns a shallow copy of the flows
func (s *Socket) Flows() []*Flow {
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
		s.PID = pid
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
