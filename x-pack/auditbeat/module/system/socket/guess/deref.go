// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"encoding/hex"
	"os"
	"strconv"
	"syscall"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v8/x-pack/auditbeat/tracing"
)

/*
	This is not an actual guess but a helper to check if the kernel kprobe
	subsystem returns garbage after dereferencing a null pointer.

	This code is run when the AUDITBEAT_SYSTEM_SOCKET_CHECK_DEREF environment
    variable is set to a value greater than 0. When set, it will run the given
    number of times, print the hexdump to the debug logs if non-zero memory is
    found and set the NULL_PTR_DEREF_IS_OK (bool) variable.
*/

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessDeref{} }); err != nil {
		panic(err)
	}
}

const (
	flagName = "NULL_PTR_DEREF_IS_OK"
	envVar   = "AUDITBEAT_SYSTEM_SOCKET_CHECK_DEREF"
)

type guessDeref struct {
	ctx     Context
	tries   int
	garbage bool
}

// Condition allows the guess to run if the environment variable is set to a
// decimal value greater than zero.
func (g *guessDeref) Condition(ctx Context) (run bool, err error) {
	v := os.Getenv(envVar)
	if v == "" {
		return false, nil
	}
	if g.tries, err = strconv.Atoi(v); err != nil || g.tries <= 0 {
		return false, nil
	}
	return true, nil
}

// Name of this guess.
func (g *guessDeref) Name() string {
	return "guess_deref"
}

// Provides returns the names of discovered variables.
func (g *guessDeref) Provides() []string {
	return []string{
		flagName,
	}
}

// Requires declares the variables required to run this guess.
func (g *guessDeref) Requires() []string {
	return []string{
		"SYS_UNAME",
		"SYS_P1",
	}
}

// Probes returns a kprobe on uname() that dumps the first bytes
// pointed to by its first parameter.
func (g *guessDeref) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKProbe,
				Name:      "guess_null_ptr_deref",
				Address:   "{{.SYS_UNAME}}",
				Fetchargs: helper.MakeMemoryDump("{{.SYS_P1}}", 0, credDumpBytes),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessDeref) Prepare(ctx Context) error {
	g.ctx = ctx
	return nil
}

// Terminate is a no-op.
func (g *guessDeref) Terminate() error {
	return nil
}

// MaxRepeats returns the configured number of repeats.
func (g *guessDeref) MaxRepeats() int {
	return g.tries
}

// Extract receives the memory read through a null pointer and checks if it's
// zero or garbage.
func (g *guessDeref) Extract(ev interface{}) (common.MapStr, bool) {
	raw := ev.([]byte)
	if len(raw) != credDumpBytes {
		return nil, false
	}
	for _, val := range raw {
		if val != 0 {
			g.ctx.Log.Errorf("Found non-zero memory:\n%s", hex.Dump(raw))
			g.garbage = true
			break
		}
	}
	// Repeat until completed all tries
	if g.tries--; g.tries > 0 {
		return nil, true
	}
	return common.MapStr{
		flagName: !g.garbage,
	}, true
}

// Trigger invokes the uname syscall with a null parameter.
func (g *guessDeref) Trigger() error {
	var ptr *syscall.Utsname
	syscall.Uname(ptr)
	return nil
}
