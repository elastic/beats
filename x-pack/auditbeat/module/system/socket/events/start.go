// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"golang.org/x/sys/unix"
)

type InetCreateCall struct {
	Meta  tracing.Metadata `kprobe:"metadata"`
	Proto int32            `kprobe:"proto"`
}

// String returns a representation of the event.
func (e *InetCreateCall) String() string {
	return fmt.Sprintf("%s inet_create(proto=%d)", header(e.Meta), e.Proto)
}

// Update the state with the contents of this event.
func (e *InetCreateCall) Update(s *state.State) {
	if e.Proto == 0 || e.Proto == unix.IPPROTO_TCP || e.Proto == unix.IPPROTO_UDP {
		s.PushThreadEvent(e.Meta.TID, e)
	}
}

type SockInitDataCall struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Socket uintptr          `kprobe:"sock"`
}

// String returns a representation of the event.
func (e *SockInitDataCall) String() string {
	return fmt.Sprintf("%s sock_init_data(sock=0x%x)", header(e.Meta), e.Socket)
}

// Update the state with the contents of this event.
func (e *SockInitDataCall) Update(s *state.State) {
	if event := s.PopThreadEvent(e.Meta.TID); event != nil {
		// Only track socks created by inet_create / inet6_create
		if call, ok := event.(*InetCreateCall); ok {
			s.UpdateFlow(state.NewFlow(
				e.Socket,
				e.Meta.PID,
				0,
				uint16(call.Proto),
				e.Meta.Timestamp,
				nil,
				nil,
			).SetCreated(e.Meta.Timestamp).MarkComplete())
		}
	}
}

// Process Start

type ExecveCall struct {
	Meta tracing.Metadata           `kprobe:"metadata"`
	Path [common.MaxProgArgLen]byte `kprobe:"path,greedy"`
	// extra ptr is to detect if there are more than maxProcArgs arguments
	Ptrs   [common.MaxProgArgs + 1]uintptr `kprobe:"argptrs,greedy"`
	Param0 [common.MaxProgArgLen]byte      `kprobe:"param0,greedy"`
	Param1 [common.MaxProgArgLen]byte      `kprobe:"param1,greedy"`
	Param2 [common.MaxProgArgLen]byte      `kprobe:"param2,greedy"`
	Param3 [common.MaxProgArgLen]byte      `kprobe:"param3,greedy"`
	Param4 [common.MaxProgArgLen]byte      `kprobe:"param4,greedy"`

	process *state.Process
}

func (e *ExecveCall) AddProcess() {
	if e.process != nil {
		return
	}

	params := [common.MaxProgArgs][]byte{
		e.Param0[:],
		e.Param1[:],
		e.Param2[:],
		e.Param3[:],
		e.Param4[:],
	}

	argc := 0
	for argc <= common.MaxProgArgs {
		if e.Ptrs[argc] == 0 {
			break
		}
		argc++
	}
	args := make([]string, argc)
	if argc > common.MaxProgArgs {
		argc = common.MaxProgArgs
		args[argc] = "..."
	}
	for i := 0; i < argc; i++ {
		args[i] = readCString(params[i])
	}

	path := readCString(e.Path[:])

	e.process = state.CreateProcess(
		e.Meta.PID,
		path,
		filepath.Base(path),
		e.Meta.Timestamp,
		args,
	)
}

// String returns a representation of the event.
func (e *ExecveCall) String() string {
	e.AddProcess()
	args := e.process.Args()

	messages := make([]string, len(args))
	for idx, val := range args {
		messages[idx] = fmt.Sprintf("arg%d='%s'", idx, val)
	}

	return fmt.Sprintf("%s execve(name='%s', path='%s', %s)", header(e.Meta), e.process.Name(), e.process.Path(), strings.Join(messages, " "))
}

// Update the state with the contents of this event.
func (e *ExecveCall) Update(s *state.State) {
	e.AddProcess()
	s.PushThreadEvent(e.Meta.TID, e)
}

type CommitCredsCall struct {
	Meta tracing.Metadata `kprobe:"metadata"`
	UID  uint32           `kprobe:"uid"`
	GID  uint32           `kprobe:"gid"`
	EUID uint32           `kprobe:"euid"`
	EGID uint32           `kprobe:"egid"`
}

// String returns a representation of the event.
func (e *CommitCredsCall) String() string {
	return fmt.Sprintf("%s commit_creds(uid=%d, gid=%d, euid=%d, egid=%d)",
		header(e.Meta),
		e.UID, e.GID, e.EUID, e.EGID)
}

// Update the state with the contents of this event.
func (e *CommitCredsCall) Update(s *state.State) {
	if event := s.PopThreadEvent(e.Meta.TID); event != nil {
		if call, ok := event.(*ExecveCall); ok {
			// Only inspect commit_creds() calls that happen in the context
			// of an execve call. Enrich the process with user information.
			if call.process != nil {
				call.process.SetCreds(e.UID, e.GID, e.EUID, e.EGID)
			}
			// Re-install the information after enrichment so that execveRet
			// can access it.
			s.PushThreadEvent(e.Meta.TID, event)
		}
	}
}

type ExecveReturn struct {
	Meta   tracing.Metadata `kprobe:"metadata"`
	Retval int32            `kprobe:"retval"`
}

// String returns a representation of the event.
func (e *ExecveReturn) String() string {
	return fmt.Sprintf("%s <- execve %s", header(e.Meta), kernErrorDesc(e.Retval))
}

// Update the state with the contents of this event.
func (e *ExecveReturn) Update(s *state.State) {
	if event := s.PopThreadEvent(e.Meta.TID); event != nil {
		if call, ok := event.(*ExecveCall); ok {
			if e.Retval >= 0 {
				s.ProcessStart(call.process)
			}
		}
	}
}

func readCString(buf []byte) string {
	if pos := bytes.IndexByte(buf, 0); pos != -1 {
		return string(buf[:pos])
	}
	return string(buf) + " ..."
}
