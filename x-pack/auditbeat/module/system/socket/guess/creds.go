// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/menderesk/beats/v7/x-pack/auditbeat/tracing"
)

/*
	creds guess discovers the offsets of (E)UID/(E)GID fields within a
    struct cred (defined in {linux}/include/linux.cred.h):
		struct cred {
			atomic_t	usage;
		#ifdef CONFIG_DEBUG_CREDENTIALS
			atomic_t	subscribers;
			void		*put_addr;
			unsigned	magic;
		#define CRED_MAGIC	0x43736564
		#define CRED_MAGIC_DEAD	0x44656144
		#endif
			kuid_t		uid;		// real UID of the task
			kgid_t		gid;		// real GID of the task
			kuid_t		suid;		// saved UID of the task
			kgid_t		sgid;		// saved GID of the task
			kuid_t		euid;		// effective UID of the task
			kgid_t		egid;		// effective GID of the task

	The output of this probe is a series of offsets within this structure:
		"STRUCT_CRED_UID": 4
		"STRUCT_CRED_GID": 8
		"STRUCT_CRED_EUID": 20
		"STRUCT_CRED_EGID": 24
*/

// This should be multiple of 8 enough to fit up to egid on a struct cred.
const (
	credDumpBytes  = 8 * 16
	credDebugMagic = 0x43736564
)

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessStructCreds{} }); err != nil {
		panic(err)
	}
}

type guessStructCreds struct{}

// Name of this guess.
func (g *guessStructCreds) Name() string {
	return "guess_struct_creds"
}

// Provides returns the names of discovered variables.
func (g *guessStructCreds) Provides() []string {
	return []string{
		"STRUCT_CRED_UID",
		"STRUCT_CRED_GID",
		"STRUCT_CRED_EUID",
		"STRUCT_CRED_EGID",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessStructCreds) Requires() []string {
	return []string{
		"RET",
	}
}

// Probes returns a kretprobe on prepare_creds that dumps the first bytes
// pointed to by the return value, which is a struct cred.
func (g *guessStructCreds) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Type:      tracing.TypeKRetProbe,
				Name:      "guess_struct_creds",
				Address:   "prepare_creds",
				Fetchargs: helper.MakeMemoryDump("{{.RET}}", 0, credDumpBytes),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessStructCreds) Prepare(Context) error {
	return nil
}

// Terminate is a no-op.
func (g *guessStructCreds) Terminate() error {
	return nil
}

// Extract receives the struct cred dump and discovers the offsets.
func (g *guessStructCreds) Extract(ev interface{}) (common.MapStr, bool) {
	raw := ev.([]byte)
	if len(raw) != credDumpBytes {
		return nil, false
	}
	const numInt32 = credDumpBytes / 4
	ptr := (*[numInt32]uint32)(unsafe.Pointer(&raw[0]))
	// default struct cred only has one int32 field before credentials
	offset := 4
	for i := 1; i < (numInt32 - 2); i++ {
		if ptr[i] == credDebugMagic {
			// Current kernel has been compiled with CONFIG_DEBUG_CREDENTIALS
			offset = 4 * (i + 1)
			break
		}
	}
	// There check is not so useful because most uid values will be zero
	// when this runs
	if ptr[offset/4] != uint32(os.Getuid()) ||
		ptr[offset/4+1] != uint32(os.Getgid()) {
		return nil, false
	}
	return common.MapStr{
		"STRUCT_CRED_UID":  offset,
		"STRUCT_CRED_GID":  offset + 4,
		"STRUCT_CRED_EUID": offset + 16,
		"STRUCT_CRED_EGID": offset + 20,
	}, true
}

// Trigger invokes the SYS_ACCESS syscall:
//	  int access(const char *pathname, int mode);
// The function call will return an error due to path being NULL, but it will
// have invoked prepare_creds before argument validation.
func (g *guessStructCreds) Trigger() error {
	syscall.Syscall(unix.SYS_ACCESS, 0, 0, 0)
	return nil
}
