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

// Guess the position of size parameter in tcp_sendmsg.
// It can be at position 3 (4.x) or 4 (2.x/3.x).
//
// Do a send(...) of a certain size and expect either arg3 is the msg length
// or arg3 is a pointer and arg4 is the length.
//
// Output:
//  TCP_SENDMSG_LEN  : +4(%sp)

func init() {
	if err := Registry.AddGuess(func() Guesser { return &guessTCPSendMsg{} }); err != nil {
		panic(err)
	}
}

type tcpSendMsgArgCountGuess struct {
	Param3 uint `kprobe:"c"`
	Param4 uint `kprobe:"d"`
}

type guessTCPSendMsg struct {
	ctx     Context
	cs      inetClientServer
	written int
}

// Name of this guess.
func (g *guessTCPSendMsg) Name() string {
	return "tcp_sendmsg_guess"
}

// Provides returns the list of variables discovered.
func (g *guessTCPSendMsg) Provides() []string {
	return []string{
		"TCP_SENDMSG_LEN",
	}
}

// Requires declares the variables required to run this guess.
func (g *guessTCPSendMsg) Requires() []string {
	return []string{
		"P3",
		"P4",
	}
}

// Probes returns a kprobe on tcp_sendmsg that fetches args 3 and 4.
func (g *guessTCPSendMsg) Probes() ([]helper.ProbeDef, error) {
	return []helper.ProbeDef{
		{
			Probe: tracing.Probe{
				Name:      "tcp_sendmsg_argcount_guess",
				Address:   "tcp_sendmsg",
				Fetchargs: "c={{.P3}} d={{.P4}}",
			},
			Decoder: helper.NewStructDecoder(func() interface{} { return new(tcpSendMsgArgCountGuess) }),
		},
	}, nil
}

// Prepare creates a TCP client-server.
func (g *guessTCPSendMsg) Prepare(ctx Context) error {
	g.ctx = ctx
	return g.cs.SetupTCP()
}

// Terminate cleans up the client-server.
func (g *guessTCPSendMsg) Terminate() error {
	return g.cs.Cleanup()
}

// Trigger writes from client to server, causing tcp_sendmsg to be called.
func (g *guessTCPSendMsg) Trigger() (err error) {
	g.written, err = unix.Write(g.cs.client, []byte("Hello World!\n"))
	return err
}

// Extract receives the arguments from the tcp_sendmsg call and checks
// which one contains the number of bytes written by trigger.
func (g *guessTCPSendMsg) Extract(ev interface{}) (mapstr.M, bool) {
	event := ev.(*tcpSendMsgArgCountGuess)
	if g.written <= 0 {
		g.ctx.Log.Errorf("write failed for guess")
	}

	var lenParam string
	switch {
	case event.Param3 == uint(g.written):
		// Linux ~4.15
		lenParam = g.ctx.Vars["P3"].(string)

	case event.Param4 == uint(g.written):
		// Older linux
		lenParam = g.ctx.Vars["P4"].(string)
	default:
		return nil, false
	}
	return mapstr.M{
		"TCP_SENDMSG_LEN": lenParam,
	}, true
}
