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
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

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

	process *common.Process
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

	e.process = common.CreateProcess(
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
	args := e.process.Args

	messages := make([]string, len(args))
	for idx, val := range args {
		messages[idx] = fmt.Sprintf("arg%d='%s'", idx, val)
	}

	return fmt.Sprintf("%s execve(name='%s', path='%s', %s)", header(e.Meta), e.process.Name, e.process.Path, strings.Join(messages, " "))
}

// Update the state with the contents of this event.
func (e *ExecveCall) Update(s common.EventTracker) {
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
	return fmt.Sprintf("%s commit_creds(uid=%d, gid=%d, euid=%d, egid=%d)", header(e.Meta), e.UID, e.GID, e.EUID, e.EGID)
}

// Update the state with the contents of this event.
func (e *CommitCredsCall) Update(s common.EventTracker) {
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
func (e *ExecveReturn) Update(s common.EventTracker) {
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
