// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package procfsprovider

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	syscallField = "auditd.data.syscall"
)

type prvdr struct {
	ctx      context.Context
	logger   *logp.Logger
	db       *processdb.DB
	reader   procfs.Reader
	pidField string
}

// NewProvider returns a new instance of procfsprovider.
func NewProvider(ctx context.Context, logger *logp.Logger, db *processdb.DB, reader procfs.Reader, pidField string) (provider.Provider, error) {
	return prvdr{
		ctx:      ctx,
		logger:   logger,
		db:       db,
		reader:   reader,
		pidField: pidField,
	}, nil
}

// GetProcess is not implemented in this provider.
// This provider adds to the processdb, and process information is retrieved from the DB, not directly from the provider
func (p prvdr) GetProcess(pid uint32) (*types.Process, error) {
	return nil, fmt.Errorf("not implemented")
}

// Sync updates the process information database using on the syscall event data and by scraping procfs.
// As process information will not be available in procfs after a process has exited, the provider is susceptible to missing information in short-lived events.
func (p prvdr) Sync(ev *beat.Event, pid uint32) error {
	syscall, err := ev.GetValue(syscallField)
	if err != nil {
		return fmt.Errorf("event not supported, no syscall data")
	}

	switch syscall {
	case "execveat", "execve":
		pe := types.ProcessExecEvent{}
		procInfo, err := p.reader.GetProcess(pid)
		if err == nil {
			pe.PIDs = procInfo.PIDs
			pe.Creds = procInfo.Creds
			pe.CTTY = procInfo.CTTY
			pe.CWD = procInfo.Cwd
			pe.Argv = procInfo.Argv
			pe.Env = procInfo.Env
			pe.Filename = procInfo.Filename
		} else {
			p.logger.Debugw("couldn't get process info from proc for pid", "pid", pid, "error", err)
			// If process info couldn't be taken from procfs, populate with as much info as
			// possible from the event
			pe.ProcfsLookupFail = true
			pe.PIDs.Tgid = pid
			var intr interface{}
			var i int
			var ok bool
			var parent types.Process
			intr, err := ev.Fields.GetValue("process.parent.pid")
			if err != nil {
				goto out
			}
			if i, ok = intr.(int); !ok {
				goto out
			}
			pe.PIDs.Ppid = uint32(i)

			parent, err = p.db.GetProcess(pe.PIDs.Ppid)
			if err != nil {
				goto out
			}
			pe.PIDs.Sid = parent.SessionLeader.PID

			intr, err = ev.Fields.GetValue("process.working_directory")
			if err != nil {
				goto out
			}
			if str, ok := intr.(string); ok {
				pe.CWD = str
			} else {
				goto out
			}
		out:
		}
		p.db.InsertExec(pe)
		if err != nil {
			return fmt.Errorf("insert exec to db: %w", err)
		}
	case "exit_group":
		pe := types.ProcessExitEvent{
			PIDs: types.PIDInfo{
				Tgid: pid,
			},
		}
		p.db.InsertExit(pe)
	case "setsid":
		intr, err := ev.Fields.GetValue("auditd.result")
		if err != nil {
			return fmt.Errorf("syscall exit value not found")
		}
		result, ok := intr.(string)
		if !ok {
			return fmt.Errorf("\"auditd.result\" not string")
		}
		if result == "success" {
			setsid_ev := types.ProcessSetsidEvent{
				PIDs: types.PIDInfo{
					Tgid: pid,
					Sid:  pid,
				},
			}
			p.db.InsertSetsid(setsid_ev)
		}
	}
	return nil
}
