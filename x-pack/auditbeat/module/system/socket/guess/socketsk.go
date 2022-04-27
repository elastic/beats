// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

// Guess the offset of (struct socket*)->sk (type struct sock*)
// This helps monitor functions that receive a socket* but what we really
// want is a sock*.
//
// 1. Creates a socket, triggering sock_init_data(socket* a, sock* b)
// 2. Closes the socket, triggering inet_release(socket *a) where a->sk == b
//
// Just the first call isn't enough because (socket*)->sk is NULL at that time.
//
// Output:
//  "SOCKET_SOCK": 32

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessSocketSock{} }); err != nil {
		panic(err)
	}
}

type sockEvent struct {
	Sock uintptr `kprobe:"sock"`
}

type guessSocketSock struct {
	ctx      Context
	initData *sockEvent
}

// Name of this guess.
func (g *guessSocketSock) Name() string {
	return "guess_struct_socket_sk"
}

// Provides returns the list of variables discovered.
func (g *guessSocketSock) Provides() []string {
	return []string{
		"SOCKET_SOCK",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessSocketSock) Requires() []string {
	return []string{
		"P1",
		"P2",
	}
}

// Probes returns two probes:
// - sock_init_data, fetching its 1st argument.
// - inet_release, dumping its only argument.
func (g *guessSocketSock) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "struct_socket_guess",
				Address:   "sock_init_data",
				Fetchargs: "sock={{.P2}}",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(sockEvent) }),
		},

		{
			Probe: tracing.Probe{
				Name:      "struct_socket_guess2",
				Address:   "inet_release",
				Fetchargs: helper.MakeMemoryDump("{{.P1}}", 0, 128),
			},
			Decoder: tracing.NewDumpDecoder,
		},
	}, nil
}

// Prepare is a no-op.
func (g *guessSocketSock) Prepare(ctx Context) error {
	g.ctx = ctx
	return nil
}

// Terminate is a no-op.
func (g *guessSocketSock) Terminate() error {
	return nil
}

// Trigger allocates and then releases a socket.
func (g *guessSocketSock) Trigger() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		return err
	}
	unix.Close(fd)
	return nil
}

// Extract first receives the sock* from sock_init_data, then uses it to
// scan the dump from inet_release.
func (g *guessSocketSock) Extract(ev interface{}) (mapstr.M, bool) {
	if v, ok := ev.(*sockEvent); ok {
		if g.initData != nil {
			return nil, false
		}
		g.initData = v
		return nil, false
	}

	dump := ev.([]byte)
	if g.initData.Sock == 0 {
		return nil, false
	}

	const ptrLen = int(sizeOfPtr)
	sockBuf := (*[ptrLen]byte)(unsafe.Pointer(&g.initData.Sock))[:]

	off := indexAligned(dump, sockBuf, 0, ptrLen)
	if off == -1 {
		return nil, false
	}

	return mapstr.M{
		"SOCKET_SOCK": off,
	}, true
}
