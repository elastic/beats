// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/auditbeat/tracing"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Guess how to get a struct sock* from tcp_sendmsg parameters. It can be:
// - param #2 (3.x)
// - param #1 (4.x).
// - indirect through a struct socket* at param #2 (2.x).
//
// Do a send(...) to a known address and try to find the destination address
// from the sock*
//
// Output:
//  TCP_SENDMSG_SOCK  : %di

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessTcpSendmsgSock{} }); err != nil {
		panic(err)
	}
}

type tcpSendMsgSockGuess struct {
	Param1   uint32 `kprobe:"p1"`
	Param2   uint32 `kprobe:"p2"`
	Indirect uint32 `kprobe:"indirect"`
}

type guessTcpSendmsgSock struct {
	ctx     Context
	cs      inetClientServer
	written int
}

// Name of this guess.
func (g *guessTcpSendmsgSock) Name() string {
	return "guess_tcp_sendmsg_sock"
}

// Provides returns the list of variables discovered.
func (g *guessTcpSendmsgSock) Provides() []string {
	return []string{
		"TCP_SENDMSG_SOCK",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessTcpSendmsgSock) Requires() []string {
	return []string{
		"INET_SOCK_RADDR",
		"SOCKET_SOCK",
		"P1",
		"P2",
	}
}

// Probes sets a kprobe on tcp_sendmsg and fetches all the possible argument
// combinations to account for all the known signatures of the function.
func (g *guessTcpSendmsgSock) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "tcp_sendmsg_sock_guess",
				Address:   "tcp_sendmsg",
				Fetchargs: "p1=+{{.INET_SOCK_RADDR}}({{.P1}}):u32 p2=+{{.INET_SOCK_RADDR}}({{.P2}}):u32 indirect=+{{.INET_SOCK_RADDR}}(+{{.SOCKET_SOCK}}({{.P2}})):u32",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpSendMsgSockGuess) }),
		},
	}, nil
}

// Prepare creates a TCP client-server.
func (g *guessTcpSendmsgSock) Prepare(ctx Context) error {
	g.ctx = ctx
	return g.cs.SetupTCP()
}

// Terminate releases the TCP client-server.
func (g *guessTcpSendmsgSock) Terminate() error {
	return g.cs.Cleanup()
}

// Trigger causes tcp_sendmsg to be called by sending data from one end
// of the connection.
func (g *guessTcpSendmsgSock) Trigger() (err error) {
	g.written, err = unix.Write(g.cs.client, []byte("Hello World!\n"))
	return err
}

// Extract checks which of the arguments to tcp_sendmsg contains the expected
// value (address of destination).
func (g *guessTcpSendmsgSock) Extract(ev interface{}) (mapstr.M, bool) {
	event := ev.(*tcpSendMsgSockGuess)
	if g.written <= 0 {
		g.ctx.Log.Errorf("write failed for guess")
		return nil, false
	}
	var param string
	wanted := tracing.MachineEndian.Uint32(g.cs.srvAddr.Addr[:])
	switch {
	case event.Indirect == wanted:
		param = fmt.Sprintf("+%d(%s)", g.ctx.Vars["SOCKET_SOCK"], g.ctx.Vars["P2"])

	case event.Param1 == wanted:
		// Linux ~4.x
		param = g.ctx.Vars["P1"].(string)

	case event.Param2 == wanted:
		// Linux ~3.x
		param = g.ctx.Vars["P2"].(string)
	default:
		return nil, false
	}
	return mapstr.M{
		"TCP_SENDMSG_SOCK": param,
	}, true
}
