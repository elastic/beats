// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

/*
	Guess the offset of local and remote IPv6 addresses on a struct inet_sock*.

	This guess is a two-in-one as the IPv6 addresses can appear in two
	different ways:

	before 3.13:

		IPv6 addresses are accessed via a pointer to an ipv6_pinfo structure:

		struct inet_sock {
			struct sock             sk;
			#if defined(CONFIG_IPV6) || defined(CONFIG_IPV6_MODULE)
			struct ipv6_pinfo       *pinet6;
			#endif
			__be32                  daddr;
			[...]

		struct ipv6_pinfo {
			struct in6_addr 	saddr;
			struct in6_addr 	rcv_saddr;
			struct in6_addr		daddr;
			[...]

		The guess finds this pointer easily because it just the field just
		before the IPv4 destination address.

	after 3.13:

		Both addresses are consecutive fields in struct sock_common:
			struct in6_addr		skc_v6_daddr;
			struct in6_addr		skc_v6_rcv_saddr;


		So what it does is:
		Probe #1 dumps the inet_sock structure, and looks for the concatenation
		of destination and source addresses.

		Probe #2 dumps the 16 bytes pointed to by the pointer-aligned address
		before all known offsets of the IPv4 destination address.

		If the addresses are found by #1 it keeps the offsets. Otherwise it
		keeps the offsets from #2.

		The results of this guess are not offsets because it needs to accommodate
		for different levels of indirection depending on the target kernel.

		A "termination" variable is used:

		In 3.13+:
			INET_SOCK_V6_RADDR_A: "+56"
			INET_SOCK_V6_TERM: ":u64"

			so that a kprobe can define:
				"{{.INET_SOCK_V6_RADDR_A}}({{.RET}}){{.INET_SOCK_V6_TERM}}"
			resulting in:
				"+56($retval):u64"

		while before 3.13 it will be:
			INET_SOCK_V6_RADDR_A: "+8(+512"
			INET_SOCK_V6_TERM: "):u64"

			and the same kprobe will result in:
				"+8(+512($retval)):u64"

		The same result could've been achieved by having the templates be
		functions that can be {{call'ed}} in the template but that cause a lot
		of trouble on its own:
			- functions need to be defined for kprobe validation
				-> defining placeholders breaks the guesses dependency resolution.
			- passing template variables as function arguments causes the
			  templates to need to be resolved recursively.

	Output:

	INET_SOCK_V6_RADDR_A: +56
	INET_SOCK_V6_RADDR_B: +64
	INET_SOCK_V6_LADDR_A: +72
	INET_SOCK_V6_LADDR_B: +80
*/

const inetSockDumpSize = 8 * 256

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessInetSockIPv6{} }); err != nil {
		panic(err)
	}
}

type guessInetSockIPv6 struct {
	ctx                    Context
	loopback               helper.IPv6Loopback
	clientAddr, serverAddr unix.SockaddrInet6
	client, server         int
	offsets                []int
	fullDump               []byte
	ptrDump                []byte
}

// Name of this guess.
func (g *guessInetSockIPv6) Name() string {
	return "guess_inet_sock_ipv6"
}

// Provides returns the names of variables provided by this guess.
func (g *guessInetSockIPv6) Provides() []string {
	return []string{
		"INET_SOCK_V6_RADDR_A",
		"INET_SOCK_V6_RADDR_B",
		"INET_SOCK_V6_LADDR_A",
		"INET_SOCK_V6_LADDR_B",
		"INET_SOCK_V6_LIMIT",
	}
}

// Requires returns the list of required variables.
func (g *guessInetSockIPv6) Requires() []string {
	return []string{
		"RET",
		"INET_SOCK_RADDR_LIST",
	}
}

// Condition allows this probe to run only when IPv6 is enabled.
func (g *guessInetSockIPv6) Condition(ctx Context) (bool, error) {
	runs, err := isIPv6Enabled(ctx.Vars)
	if err != nil {
		return false, err
	}
	if !runs {
		// Set a safe default for INET_SOCK_V6_LIMIT so that guesses
		// depending on it can run.
		ctx.Vars["INET_SOCK_V6_LIMIT"] = 2048
	}
	return runs, nil
}

// eventWrapper is used to wrap events from one of the probes for differentiation.
type eventWrapper struct {
	event interface{}
}

// decoderWrapper takes an inner decoder and wraps it so that the returned events
// are wrapped with the eventWrapper type.
type decoderWrapper struct {
	inner tracing.Decoder
}

// Decode wraps events from the inner decoder into a the wrapper type.
func (d *decoderWrapper) Decode(raw []byte, meta tracing.Metadata) (event interface{}, err error) {
	if event, err = d.inner.Decode(raw, meta); err != nil {
		return event, err
	}
	return eventWrapper{event}, nil
}

// Probes returns a kretprobe in inet_csk_accept that dumps the memory pointed
// to by the return value (an inet_sock*) and a kretprobe that dumps various
// candidates for the ipv6_pinfo struct.
func (g *guessInetSockIPv6) Probes() (probes []helper.ProbeDef, err error) {
	probes = append(probes, helper.ProbeDef{
		Probe: tracing.Probe{
			Type:      tracing.TypeKRetProbe,
			Name:      "inet_sock_ipv6_guess",
			Address:   "inet_csk_accept",
			Fetchargs: helper.MakeMemoryDump("{{.RET}}", 0, inetSockDumpSize),
		},
		Decoder: tracing.NewDumpDecoder,
	})
	raddrOffsets := g.offsets
	g.offsets = nil
	const sizePtr = int(sizeOfPtr)
	var fetch []string
	for _, off := range raddrOffsets {
		// the pointer we're looking for is after a field of sizeof(struct sock)
		if off < sizePtr*2 {
			continue
		}
		// This is aligning to the nearest offset aligned to sizePtr and smaller
		// than off.
		off = alignTo(off-2*sizePtr+1, sizePtr)
		g.offsets = append(g.offsets, off)
		// dumps the rcv_saddr field of struct ipv6_pinfo
		fetch = append(fetch, fmt.Sprintf("+16(+%d({{.RET}})):u64 +24(+%d({{.RET}})):u64", off, off))
	}
	probes = append(probes, helper.ProbeDef{
		Probe: tracing.Probe{
			Type:      tracing.TypeKRetProbe,
			Name:      "inet_sock_ipv6_guess2",
			Address:   "inet_csk_accept",
			Fetchargs: strings.Join(fetch, " "),
		},
		Decoder: func(desc tracing.ProbeFormat) (decoder tracing.Decoder, err error) {
			if decoder, err = tracing.NewDumpDecoder(desc); err != nil {
				return nil, err
			}
			return &decoderWrapper{decoder}, nil
		},
	})
	return probes, nil
}

// Prepare creates an IPv6 client/server bound to loopback.
// Unlike with IPv4, it's not possible to bind to a random IP in the IPv6
// loopback as it is ::1/128 by default. Thus it's necessary to add temporary
// random addresses in the fd00::/8 reserved network to the loopback device.
func (g *guessInetSockIPv6) Prepare(ctx Context) (err error) {
	g.ctx = ctx
	g.offsets, err = getListField(g.ctx.Vars, "INET_SOCK_RADDR_LIST")
	if err != nil {
		return err
	}
	g.loopback, err = helper.NewIPv6Loopback()
	if err != nil {
		return fmt.Errorf("detect IPv6 loopback failed: %w", err)
	}
	defer func() {
		if err != nil {
			g.loopback.Cleanup()
		}
	}()
	clientIP, err := g.loopback.AddRandomAddress()
	if err != nil {
		return fmt.Errorf("failed adding first device address: %w", err)
	}
	serverIP, err := g.loopback.AddRandomAddress()
	if err != nil {
		return fmt.Errorf("failed adding second device address: %w", err)
	}
	copy(g.clientAddr.Addr[:], clientIP)
	copy(g.serverAddr.Addr[:], serverIP)

	if g.client, g.clientAddr, err = createSocket6WithProto(unix.SOCK_STREAM, g.clientAddr); err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}
	if g.server, g.serverAddr, err = createSocket6WithProto(unix.SOCK_STREAM, g.serverAddr); err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	if err = unix.Listen(g.server, 1); err != nil {
		return fmt.Errorf("error in listen: %w", err)
	}
	return nil
}

// Trigger connects the client to the server, causing an inet_csk_accept call.
func (g *guessInetSockIPv6) Trigger() error {
	if err := unix.Connect(g.client, &g.serverAddr); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	fd, _, err := unix.Accept(g.server)
	if err != nil {
		return fmt.Errorf("accept failed: %w", err)
	}
	unix.Close(fd)
	return nil
}

// Extract scans stores the events from the two different kprobes and then
// looks for the result in one of them.
func (g *guessInetSockIPv6) Extract(event interface{}) (common.MapStr, bool) {
	if w, ok := event.(eventWrapper); ok {
		g.ptrDump = w.event.([]byte)
	} else {
		g.fullDump = event.([]byte)
	}
	if g.ptrDump == nil || g.fullDump == nil {
		return nil, false
	}

	result, ok := g.searchStructSock(g.fullDump)
	if !ok {
		result, ok = g.searchIPv6PInfo(g.ptrDump)
	}
	return result, ok
}

func (g *guessInetSockIPv6) searchStructSock(raw []byte) (common.MapStr, bool) {
	var expected []byte
	expected = append(expected, g.clientAddr.Addr[:]...) // sck_v6_daddr
	expected = append(expected, g.serverAddr.Addr[:]...) // sck_v6_rcv_saddr
	offset := bytes.Index(raw, expected)
	if offset == -1 {
		return nil, false
	}
	return common.MapStr{
		"INET_SOCK_V6_TERM":    ":u64",
		"INET_SOCK_V6_RADDR_A": fmt.Sprintf("+%d", offset),
		"INET_SOCK_V6_RADDR_B": fmt.Sprintf("+%d", offset+8),
		"INET_SOCK_V6_LADDR_A": fmt.Sprintf("+%d", offset+16),
		"INET_SOCK_V6_LADDR_B": fmt.Sprintf("+%d", offset+24),
		"INET_SOCK_V6_LIMIT":   offset,
	}, true
}

func (g *guessInetSockIPv6) searchIPv6PInfo(raw []byte) (common.MapStr, bool) {
	offset := bytes.Index(raw, g.serverAddr.Addr[:])
	if offset == -1 {
		return nil, false
	}
	idx := offset / 16 // length of IPv6 address
	if idx >= len(g.offsets) {
		return nil, false
	}
	off := g.offsets[idx]
	return common.MapStr{
		"INET_SOCK_V6_TERM":    "):u64",
		"INET_SOCK_V6_RADDR_A": fmt.Sprintf("+%d(+%d", 32, off),
		"INET_SOCK_V6_RADDR_B": fmt.Sprintf("+%d(+%d", 40, off),
		"INET_SOCK_V6_LADDR_A": fmt.Sprintf("+%d(+%d", 16, off),
		"INET_SOCK_V6_LADDR_B": fmt.Sprintf("+%d(+%d", 24, off),
		"INET_SOCK_V6_LIMIT":   off,
	}, true
}

// Terminate closes the client/server and releases the random IPs from loopback.
func (g *guessInetSockIPv6) Terminate() error {
	unix.Close(g.client)
	unix.Close(g.server)
	if err := g.loopback.Cleanup(); err != nil {
		return err
	}
	return nil
}
