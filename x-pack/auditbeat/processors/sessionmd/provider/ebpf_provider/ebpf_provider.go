// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package ebpf_provider

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/ebpf"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/ebpfevents"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	name      = "add_session_metadata"
	eventMask = ebpf.EventMask(ebpfevents.EventTypeProcessFork | ebpfevents.EventTypeProcessExec | ebpfevents.EventTypeProcessExit)
)

type prvdr struct {
	ctx    context.Context
	logger *logp.Logger
	db     *processdb.DB
}

func NewProvider(ctx context.Context, logger *logp.Logger, db *processdb.DB) (provider.Provider, error) {
	p := prvdr{
		ctx:    ctx,
		logger: logger,
		db:     db,
	}

	w, err := ebpf.GetWatcher()
	if err != nil {
		return nil, fmt.Errorf("get ebpf watcher: %w", err)
	}

	records := w.Subscribe(name, eventMask)

	go func(logger logp.Logger) {
		for {
			r := <-records
			if r.Error != nil {
				logger.Warnw("received error from the ebpf subscription", "error", err)
				continue
			}
			if r.Event == nil {
				continue
			}
			ev := r.Event
			switch ev.Type {
			case ebpfevents.EventTypeProcessFork:
				body, ok := ev.Body.(*ebpfevents.ProcessFork)
				if !ok {
					logger.Errorf("unexpected event body, got %T", ev.Body)
					continue
				}
				pe := types.ProcessForkEvent{
					ParentPIDs: types.PIDInfo{
						Tid:         body.ParentPids.Tid,
						Tgid:        body.ParentPids.Tgid,
						Ppid:        body.ParentPids.Ppid,
						Pgid:        body.ParentPids.Pgid,
						Sid:         body.ParentPids.Sid,
						StartTimeNS: body.ParentPids.StartTimeNs,
					},
					ChildPIDs: types.PIDInfo{
						Tid:         body.ChildPids.Tid,
						Tgid:        body.ChildPids.Tgid,
						Ppid:        body.ChildPids.Ppid,
						Pgid:        body.ChildPids.Pgid,
						Sid:         body.ChildPids.Sid,
						StartTimeNS: body.ChildPids.StartTimeNs,
					},
					Creds: types.CredInfo{
						Ruid:         body.Creds.Ruid,
						Rgid:         body.Creds.Rgid,
						Euid:         body.Creds.Euid,
						Egid:         body.Creds.Egid,
						Suid:         body.Creds.Suid,
						Sgid:         body.Creds.Sgid,
						CapPermitted: body.Creds.CapPermitted,
						CapEffective: body.Creds.CapEffective,
					},
				}
				p.db.InsertFork(pe)
			case ebpfevents.EventTypeProcessExec:
				body, ok := ev.Body.(*ebpfevents.ProcessExec)
				if !ok {
					logger.Errorf("unexpected event body")
					continue
				}
				pe := types.ProcessExecEvent{
					PIDs: types.PIDInfo{
						Tid:         body.Pids.Tid,
						Tgid:        body.Pids.Tgid,
						Ppid:        body.Pids.Ppid,
						Pgid:        body.Pids.Pgid,
						Sid:         body.Pids.Sid,
						StartTimeNS: body.Pids.StartTimeNs,
					},
					Creds: types.CredInfo{
						Ruid:         body.Creds.Ruid,
						Rgid:         body.Creds.Rgid,
						Euid:         body.Creds.Euid,
						Egid:         body.Creds.Egid,
						Suid:         body.Creds.Suid,
						Sgid:         body.Creds.Sgid,
						CapPermitted: body.Creds.CapPermitted,
						CapEffective: body.Creds.CapEffective,
					},
					CTTY: types.TTYDev{
						Major: body.CTTY.Major,
						Minor: body.CTTY.Minor,
					},
					CWD:      body.Cwd,
					Argv:     body.Argv,
					Env:      body.Env,
					Filename: body.Filename,
				}
				p.db.InsertExec(pe)
			case ebpfevents.EventTypeProcessExit:
				body, ok := ev.Body.(*ebpfevents.ProcessExit)
				if !ok {
					logger.Errorf("unexpected event body")
					continue
				}
				pe := types.ProcessExitEvent{
					PIDs: types.PIDInfo{
						Tid:         body.Pids.Tid,
						Tgid:        body.Pids.Tgid,
						Ppid:        body.Pids.Ppid,
						Pgid:        body.Pids.Pgid,
						Sid:         body.Pids.Sid,
						StartTimeNS: body.Pids.StartTimeNs,
					},
					ExitCode: body.ExitCode,
				}
				p.db.InsertExit(pe)
			}
		}
	}(*p.logger)

	return &p, nil
}

func (s prvdr) UpdateDB(ev *beat.Event) error {
	// no-op for ebpf, DB is updated from pushed ebpf events
	return nil
}
