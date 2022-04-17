// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"errors"

	"golang.org/x/sys/unix"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/menderesk/beats/v7/x-pack/auditbeat/tracing"
)

/*
	This guess scans a struct inet_sock* for the address family field, which
	is used to tell if the sock is IPv4 (AF_INET=2) or IPv6 (AF_INET6=10).

	This is actually one of the first fields of struct sock_common, which is
	embedded at the start of struct inet_sock:

	struct sock_common {
		union {
			struct hlist_node       skc_node;
			struct hlist_nulls_node skc_nulls_node;
		};
		atomic_t                skc_refcnt;

		unsigned int            skc_hash;
		unsigned short          skc_family; // <-- here
	[...]

	As it is a short and values are prone to appear in many difference offsets
	by pure chance, this guess is repeated multiple times alternating between
	AF_INET and AF_INET6, and results in the only offset that had the expected
	value after all the runs.

	Output:
		INET_SOCK_AF: 16
*/

const inetSockAfDumpSize = 8 * 16

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessInetSockFamily{} }); err != nil {
		panic(err)
	}
}

type guessInetSockFamily struct {
	ctx     Context
	family  int
	limit   int
	canIPv6 bool
}

// Name of this guess.
func (g *guessInetSockFamily) Name() string {
	return "guess_inet_sock_af"
}

// Provides returns the list of variables discovered.
func (g *guessInetSockFamily) Provides() []string {
	return []string{
		"INET_SOCK_AF",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessInetSockFamily) Requires() []string {
	return []string{
		"SOCKET_SOCK",
		"INET_SOCK_V6_LIMIT",
		"P1",
	}
}

// Probes returns a kprobe on inet_release which has a struct socket* as
// single argument. Returns a dump of the (struct socket*)->sk field, which is
// a struct inet_sock* for INET/INET6.
func (g *guessInetSockFamily) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "inet_sock_af_guess",
				Address:   "inet_release",
				Fetchargs: helper.MakeMemoryDump("+{{.SOCKET_SOCK}}({{.P1}})", 0, inetSockAfDumpSize),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessInetSockFamily) Prepare(ctx Context) error {
	g.ctx = ctx
	var ok bool
	// limit is used as a reference point within struct sock_common to know where
	// to stop looking for the skc_family field, as limit is part of the fields
	// in inet_sock that are past struct sock.
	if g.limit, ok = g.ctx.Vars["INET_SOCK_V6_LIMIT"].(int); !ok {
		return errors.New("required variable INET_SOCK_V6_LIMIT not found")
	}
	// check that this system can create AF_INET6 sockets. Otherwise revert to
	// using AF_INET only.
	fd, err := unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, 0)
	if g.canIPv6 = err == nil; g.canIPv6 {
		unix.Close(fd)
	}
	return nil
}

// Terminate is a no-op.
func (g *guessInetSockFamily) Terminate() error {
	return nil
}

// Trigger creates and then closes a socket alternating between AF_INET/AF_INET6
// on each run.
func (g *guessInetSockFamily) Trigger() error {
	if g.canIPv6 && g.family == unix.AF_INET {
		g.family = unix.AF_INET6
	} else {
		g.family = unix.AF_INET
	}
	fd, err := unix.Socket(g.family, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	unix.Close(fd)
	return nil
}

// Extract scans the struct inet_sock* memory for the current address family value.
func (g *guessInetSockFamily) Extract(event interface{}) (common.MapStr, bool) {
	raw := event.([]byte)
	var expected [2]byte
	var hits []int
	tracing.MachineEndian.PutUint16(expected[:], uint16(g.family))

	off := indexAligned(raw, expected[:], 0, 2)
	for off != -1 && off < g.limit {
		hits = append(hits, off)
		off = indexAligned(raw, expected[:], off+2, 2)
	}
	if len(hits) == 0 {
		return nil, false
	}
	return common.MapStr{
		"INET_SOCK_AF": hits,
	}, true
}

// NumRepeats returns how many times to repeat this guess.
func (g *guessInetSockFamily) NumRepeats() int {
	return 10
}

// Reduce takes the output of the multiple runs and consolidates a single result.
func (g *guessInetSockFamily) Reduce(results []common.MapStr) (result common.MapStr, err error) {
	if result, err = consolidate(results); err != nil {
		return nil, err
	}
	list, err := getListField(result, "INET_SOCK_AF")
	if err != nil {
		return nil, err
	}
	result["INET_SOCK_AF"] = list[0]
	return result, nil
}
