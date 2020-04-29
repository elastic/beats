// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package perf

import (
	"errors"
	"fmt"
)

// Group configures a group of events.
type Group struct {
	// CountFormat configures the format of counts read from the event
	// leader. The Group option is set automatically.
	CountFormat CountFormat

	// Options configures options for all events in the group.
	Options Options

	// ClockID configures the clock for samples in the group.
	ClockID int32

	err             error // sticky configuration error
	attrs           []*Attr
	leaderNeedsRing bool
}

// TODO(acln): what other fields belong on Group? SampleFormat perhaps?

// Add adds events to the group, as configured by cfgs.
//
// For each Configurator, a new *Attr is created, the group-specific settings
// are applied, then Configure is called on the *Attr to produce the final
// event attributes.
func (g *Group) Add(cfgs ...Configurator) {
	for _, cfg := range cfgs {
		g.add(cfg)
	}
}

func (g *Group) add(cfg Configurator) {
	if g.err != nil {
		return
	}
	a := new(Attr)
	a.CountFormat = g.CountFormat
	a.Options = g.Options
	a.ClockID = g.ClockID
	err := cfg.Configure(a)
	if err != nil {
		g.err = err
		return
	}
	if a.Sample != 0 {
		g.leaderNeedsRing = true
	}
	g.attrs = append(g.attrs, a)
}

// Open opens all the events in the group, and returns their leader.
//
// The returned Event controls the entire group. Callers must use the
// ReadGroupCount method when reading counters from it. Closing it closes
// the entire group.
func (g *Group) Open(pid int, cpu int) (*Event, error) {
	if len(g.attrs) == 0 {
		return nil, errors.New("perf: empty event group")
	}
	if g.err != nil {
		return nil, fmt.Errorf("perf: configuration error: %v", g.err)
	}
	leaderattr := g.attrs[0]
	leaderattr.CountFormat.Group = true
	leader, err := Open(leaderattr, pid, cpu, nil)
	if err != nil {
		return nil, fmt.Errorf("perf: failed to open event leader: %v", err)
	}
	if len(g.attrs) < 2 {
		return leader, nil
	}
	if g.leaderNeedsRing {
		if err := leader.MapRing(); err != nil {
			return nil, fmt.Errorf("perf: failed to map leader ring: %v", err)
		}
	}
	for idx, attr := range g.attrs[1:] {
		follower, err := Open(attr, pid, cpu, leader)
		if err != nil {
			leader.Close()
			return nil, fmt.Errorf("perf: failed to open group event #%d (%q): %v", idx, attr.Label, err)
		}
		leader.owned = append(leader.owned, follower)
		if attr.Sample != 0 {
			if err := follower.SetOutput(leader); err != nil {
				leader.Close()
				return nil, fmt.Errorf("perf: failed to route follower %q output to leader %q (pid %d on CPU %d)", attr.Label, leaderattr.Label, pid, cpu)
			}
		}
	}
	return leader, nil
}

// A Configurator configures event attributes. Implementations should only
// set the fields they need. See (*Group).Add for more details.
type Configurator interface {
	Configure(attr *Attr) error
}

type configuratorFunc func(attr *Attr) error

func (cf configuratorFunc) Configure(attr *Attr) error { return cf(attr) }
