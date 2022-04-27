// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

/*
	Guess syscall calling convention.

	In kernel 4.17, the calling convention used to pass arguments from the
	kernel trap handler to the syscall handlers changed.
	See https://lwn.net/Articles/752422/

	Now these functions receive a single argument which is a pointer to a pt_reg
	structure.

	This detects if this new convention is in use and adjusts the syscall
	argument templates (SYS_Pn) accordingly.
*/

func init() {
	if err := Registry.AddGuess(func() Guesser {
		return &guessSyscallArgs{
			expected: [2]uintptr{^uintptr(0x11111111), ^uintptr(0x22222222)},
		}
	}); err != nil {
		panic(err)
	}
}

type syscallGuess struct {
	RegP1 uintptr `kprobe:"p1reg"`
	RegP2 uintptr `kprobe:"p2reg"`
	PtP1  uintptr `kprobe:"p1pt"`
	PtP2  uintptr `kprobe:"p2pt"`
}

type guessSyscallArgs struct {
	ctx      Context
	expected [2]uintptr
}

// Name of this guess.
func (g *guessSyscallArgs) Name() string {
	return "guess_syscall_args"
}

// Provides returns the list of variables discovered.
func (g *guessSyscallArgs) Provides() []string {
	return []string{
		"SYS_P1",
		"SYS_P2",
		"SYS_P3",
		"SYS_P4",
		"SYS_P5",
		"SYS_P6",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessSyscallArgs) Requires() []string {
	return []string{
		"SYS_GETTIMEOFDAY",
		"_SYS_P1",
		"_SYS_P2",
	}
}

// Probes sets a probe on gettimeofday syscall, and returns its first and
// second argument according to both calling conventions.
func (g *guessSyscallArgs) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "syscall_args_guess",
				Address:   "{{.SYS_GETTIMEOFDAY}}",
				Fetchargs: "p1reg={{._SYS_P1}} p2reg={{._SYS_P2}} p1pt=+0x70({{._SYS_P1}}) p2pt=+0x68({{(._SYS_P1)}})",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(syscallGuess) }),
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessSyscallArgs) Prepare(ctx Context) error {
	g.ctx = ctx
	return nil
}

// Terminate is a no-op.
func (g *guessSyscallArgs) Terminate() error {
	return nil
}

// Trigger invokes gettimeofday() with magic values as arguments 1 and 2.
func (g *guessSyscallArgs) Trigger() error {
	syscall.Syscall(unix.SYS_GETTIMEOFDAY, g.expected[0], g.expected[1], 0)
	return nil
}

// Extract check which set of kprobe arguments received the magic values.
func (g *guessSyscallArgs) Extract(ev interface{}) (mapstr.M, bool) {
	args, ok := ev.(*syscallGuess)
	if !ok {
		return nil, false
	}

	if args.RegP1 == g.expected[0] && args.RegP2 == g.expected[1] {
		// Current kernel uses the old calling convention.
		return mapstr.M{
			"SYS_P1": g.ctx.Vars["_SYS_P1"],
			"SYS_P2": g.ctx.Vars["_SYS_P2"],
			"SYS_P3": g.ctx.Vars["_SYS_P3"],
			"SYS_P4": g.ctx.Vars["_SYS_P4"],
			"SYS_P5": g.ctx.Vars["_SYS_P5"],
			"SYS_P6": g.ctx.Vars["_SYS_P6"],
		}, true
	}
	// New calling convention detected. Adjust syscall arguments to read
	// well known offsets of the pt_regs structure at position 1.
	// This hardcodes %di as arg1, which is only true for amd64, but this
	// calling convention is only used under amd64.
	if args.PtP1 == g.expected[0] && args.PtP2 == g.expected[1] {
		return mapstr.M{
			"SYS_P1": "+0x70(%di)",
			"SYS_P2": "+0x68(%di)",
			"SYS_P3": "+0x60(%di)",
			"SYS_P4": "+0x38(%di)",
			"SYS_P5": "+0x48(%di)",
			"SYS_P6": "+0x40(%di)",
		}, true
	}

	return nil, false
}
