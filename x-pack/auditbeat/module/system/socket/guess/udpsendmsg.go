// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package guess

import (
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// guess udp_sendmsg arguments:
//
//  int udp_sendmsg(struct kiocb *iocb, struct sock *sk, struct msghdr *msg,
//                  size_t len) // 2.x / 3.x
//
//  int udp_sendmsg(struct sock *sk, struct msghdr *msg, size_t len) // 4.x
//
//
// output:
//  UDP_SENDMSG_LEN: $stack4
//  UDP_SENDMSG_SOCK: $stack2
//  UDP_SENDMSG_MSG: $stack3

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessUDPSendMsg{} }); err != nil {
		panic(err)
	}
}

type udpSendMsgCountGuess struct {
	Param3 uintptr `kprobe:"c"`
	Param4 uintptr `kprobe:"d"`
}

type guessUDPSendMsg struct {
	ctx    Context
	cs     inetClientServer
	length uintptr
}

// Name of this guess.
func (g *guessUDPSendMsg) Name() string {
	return "guess_udp_sendmsg"
}

// Provides returns the list of variables discovered.
func (g *guessUDPSendMsg) Provides() []string {
	return []string{
		"UDP_SENDMSG_SOCK",
		"UDP_SENDMSG_LEN",
		"UDP_SENDMSG_MSG",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessUDPSendMsg) Requires() []string {
	return []string{
		"P3",
		"P4",
	}
}

// Probes returns a probe on udp_sendmsg that dumps all 4 possible arguments.
func (g *guessUDPSendMsg) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "udp_sendmsg_guess",
				Address:   "udp_sendmsg",
				Fetchargs: "c={{.P3}} d={{.P4}}",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(udpSendMsgCountGuess) }),
		},
	}, nil
}

func (g *guessUDPSendMsg) Prepare(ctx Context) error {
	g.ctx = ctx
	return g.cs.SetupUDP()
}

func (g *guessUDPSendMsg) Terminate() error {
	return g.cs.Cleanup()
}

func (g *guessUDPSendMsg) Extract(ev interface{}) (mapstr.M, bool) {
	if g.length == 0 {
		return nil, false
	}
	event := ev.(*udpSendMsgCountGuess)
	if event.Param3 == g.length {
		return mapstr.M{
			"UDP_SENDMSG_SOCK": g.ctx.Vars["P1"],
			"UDP_SENDMSG_MSG":  g.ctx.Vars["P2"],
			"UDP_SENDMSG_LEN":  g.ctx.Vars["P3"],
		}, true
	}
	if event.Param4 == g.length {
		return mapstr.M{
			"UDP_SENDMSG_SOCK": g.ctx.Vars["P2"],
			"UDP_SENDMSG_MSG":  g.ctx.Vars["P3"],
			"UDP_SENDMSG_LEN":  g.ctx.Vars["P4"],
		}, true
	}
	return nil, false
}

func (g *guessUDPSendMsg) Trigger() error {
	buf := []byte("Hello World!\n")
	unix.Sendto(g.cs.client, buf, unix.MSG_NOSIGNAL, &g.cs.srvAddr)
	unix.Recvfrom(g.cs.server, buf, 0)
	g.length = uintptr(len(buf))
	return nil
}
