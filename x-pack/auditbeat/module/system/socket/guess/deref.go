// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"encoding/hex"
	"os"
	"syscall"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
)

/*

 */

func init() {
	if err := Registry.AddGuess(&guessDeref{}); err != nil {
		panic(err)
	}
}

const (
	flagName = "NULL_PTR_DEREF_IS_GARBAGE"
	envVar   = "AUDITBEAT_SYSTEM_SOCKET_CHECK_DEREF"
)

type guessDeref struct {
	ctx Context
}

func (g *guessDeref) Condition(ctx Context) (bool, error) {
	v := os.Getenv(envVar)
	return v == "1", nil
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
	}
}

// Probes returns a kretprobe on prepare_creds that dumps the first bytes
// pointed to by the return value, which is a struct cred.
func (g *guessDeref) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKProbe,
				Name:      "guess_null_ptr_deref",
				Address:   "{{.SYS_UNAME}}",
				Fetchargs: helper.MakeMemoryDump("{{.P1}}", 0, credDumpBytes),
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

func (g *guessDeref) MaxRepeats() int {
	return 1000
}

// Extract receives the struct cred dump and discovers the offsets.
func (g *guessDeref) Extract(ev interface{}) (common.MapStr, bool) {
	raw := ev.([]byte)
	if len(raw) != credDumpBytes {
		return nil, false
	}
	for _, val := range raw {
		if val != 0 {
			g.ctx.Log.Errorf("Found non-empty memory:\n%s", hex.Dump(raw))
			//return nil, false
		}
	}
	return nil, true
}

// Trigger invokes the SYS_ACCESS syscall:
//	  int access(const char *pathname, int mode);
// The function call will return an error due to path being NULL, but it will
// have invoked prepare_creds before argument validation.
func (g *guessDeref) Trigger() error {
	var ptr *syscall.Utsname
	syscall.Uname(ptr)
	return nil
}
