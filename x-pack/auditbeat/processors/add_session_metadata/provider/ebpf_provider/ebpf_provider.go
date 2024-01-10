// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ebpf_provider

import (
	"context"

	"github.com/mohae/deepcopy"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/pkg/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/types"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/ebpfevents"
)

type prvdr struct {
	ctx                context.Context
	logger             logp.Logger
	db                 processdb.DB
}

func NewProvider(ctx context.Context, logger logp.Logger, db processdb.DB) (provider.Provider, error) {
	p := prvdr{
		ctx:    ctx,
		logger: logger,
		db:     db,
	}

	l, err := ebpfevents.NewLoader()
	if err != nil {
		p.logger.Errorf("loading ebpf: %w", err)
		return prvdr{}, err
	}

	events := make(chan ebpfevents.Event)
	errors := make(chan error)

	go l.EventLoop(p.ctx, events, errors)

	go func(logger logp.Logger) {
		for {
			select {
			case err := <-errors:
				logger.Errorf("recv'd error: %w", err)
				continue
			case ev := <-events:
				if err != nil {
					logger.Errorf("marshal event: %w", err)
					continue
				}
				switch ev.Type {
				case ebpfevents.EventTypeProcessFork:
					body, ok := ev.Body.(*ebpfevents.ProcessFork)
					if !ok {
						logger.Errorf("unexpected event body")
						continue
					}
					pe := types.ProcessForkEvent{
						ParentPids: types.PidInfo{
							Tid:         body.ParentPids.Tid,
							Tgid:        body.ParentPids.Tgid,
							Ppid:        body.ParentPids.Ppid,
							Pgid:        body.ParentPids.Pgid,
							Sid:         body.ParentPids.Sid,
							StartTimeNs: body.ParentPids.StartTimeNs,
						},
						ChildPids: types.PidInfo{
							Tid:         body.ChildPids.Tid,
							Tgid:        body.ChildPids.Tgid,
							Ppid:        body.ChildPids.Ppid,
							Pgid:        body.ChildPids.Pgid,
							Sid:         body.ChildPids.Sid,
							StartTimeNs: body.ChildPids.StartTimeNs,
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
						Pids: types.PidInfo{
							Tid:         body.Pids.Tid,
							Tgid:        body.Pids.Tgid,
							Ppid:        body.Pids.Ppid,
							Pgid:        body.Pids.Pgid,
							Sid:         body.Pids.Sid,
							StartTimeNs: body.Pids.StartTimeNs,
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
						CTty: types.TtyDev{
							Major: body.CTTY.Major,
							Minor: body.CTTY.Minor,
						},
						Cwd:              body.Cwd,
						Argv:             deepcopy.Copy(body.Argv).([]string),
						Env:              deepcopy.Copy(body.Env).(map[string]string),
						Filename:         body.Filename,
					}
					p.db.InsertExec(pe)
				case ebpfevents.EventTypeProcessExit:
					body, ok := ev.Body.(*ebpfevents.ProcessExit)
					if !ok {
						logger.Errorf("unexpected event body")
						continue
					}
					pe := types.ProcessExitEvent{
						Pids: types.PidInfo{
							Tid:         body.Pids.Tid,
							Tgid:        body.Pids.Tgid,
							Ppid:        body.Pids.Ppid,
							Pgid:        body.Pids.Pgid,
							Sid:         body.Pids.Sid,
							StartTimeNs: body.Pids.StartTimeNs,
						},
						ExitCode:         body.ExitCode,
					}
					p.db.InsertExit(pe)
				}
				continue
			}
		}
	}(p.logger)

	return &p, nil
}

func (s prvdr) UpdateDB(ev *beat.Event) error {
	// no-op for ebpf, DB is updated from pushed ebpf events
	return nil
}
