// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package procfs_provider

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

func NewProvider(ctx context.Context, logger *logp.Logger, db *processdb.DB, reader procfs.Reader, pidField string) (provider.Provider, error) {
	return prvdr{
		ctx:      ctx,
		logger:   logger,
		db:       db,
		reader:   reader,
		pidField: pidField,
	}, nil
}

// UpdateDB will update the process DB with process info from procfs or the event itself
func (s prvdr) UpdateDB(ev *beat.Event) error {
	pi, err := ev.Fields.GetValue(s.pidField)
	if err != nil {
		return fmt.Errorf("event not supported, no pid")
	}
	pid, ok := pi.(int)
	if !ok {
		return fmt.Errorf("pid field not int")
	}

	syscall, err := ev.GetValue(syscallField)
	if err != nil {
		return fmt.Errorf("event not supported, no syscall data")
	}

	switch syscall {
	case "execveat", "execve":
		pe := types.ProcessExecEvent{}
		proc_info, err := s.reader.GetProcess(uint32(pid))
		if err == nil {
			pe.PIDs = proc_info.PIDs
			pe.Creds = proc_info.Creds
			pe.CTTY = proc_info.CTTY
			pe.CWD = proc_info.Cwd
			pe.Argv = proc_info.Argv
			pe.Env = proc_info.Env
			pe.Filename = proc_info.Filename
		} else {
			s.logger.Errorf("get process info from proc for pid %v: %w", pid, err)
			// If process info couldn't be taken from procfs, populate with as much info as
			// possible from the event
			pe.PIDs.Tgid = uint32(pid)
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

			parent, err = s.db.GetProcess(pe.PIDs.Ppid)
			if err != nil {
				goto out
			}
			pe.PIDs.Sid = parent.SessionLeader.PID

			intr, err = ev.Fields.GetValue("process.working_directory")
			if err != nil {
				goto out
			}
			pe.CWD = intr.(string)
		out:
		}
		s.db.InsertExec(pe)
		if err != nil {
			return fmt.Errorf("insert exec to db: %w", err)
		}
	case "exit_group":
		pe := types.ProcessExitEvent{
			PIDs: types.PIDInfo{
				Tgid: uint32(pid),
			},
		}
		s.db.InsertExit(pe)
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
					Tgid: uint32(pid),
					Sid:  uint32(pid),
				},
			}
			s.db.InsertSetsid(setsid_ev)
		}
	}
	return nil
}
