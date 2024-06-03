// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package quarkprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	quark "github.com/elastic/quark/go"
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

	qq, err := quark.OpenQueue(quark.QueueAttr{
		Flags:     (quark.QQ_KPROBE),
		MaxLength: 1000,
	}, 64)
	if err != nil {
		return nil, fmt.Errorf("open queue: %v", err)
	}

	pid1 := qq.Lookup(1)
	if pid1 != nil {
		logger.Error("MWOLF: got PID!")
	}

	go func(qq *quark.Queue, logger *logp.Logger) {
		for {
			qevs, err := qq.GetEvents()
			if err != nil {
				logger.Errorf("get events from quark: %v", err)
				continue
			}
			for _, qev := range qevs {
				pr := processdb.Process{
					PIDs: types.PIDInfo{
						Tid:         qev.Pid,
						Tgid:        qev.Pid,
						Ppid:        qev.Proc.Ppid,
						Pgid:        qev.Proc.Pgid,
						Sid:         qev.Proc.Sid,
						StartTimeNS: qev.Proc.TimeBoot,
					},
					Creds: types.CredInfo{
						Ruid:         qev.Proc.Uid,
						Rgid:         qev.Proc.Gid,
						Euid:         qev.Proc.Euid,
						Egid:         qev.Proc.Egid,
						Suid:         qev.Proc.Suid,
						Sgid:         qev.Proc.Sgid,
						CapPermitted: qev.Proc.CapPermitted,
						CapEffective: qev.Proc.CapEffective,
					},
					CTTY: types.TTYDev{
						Major: uint16(qev.Proc.TtyMajor),
						Minor: uint16(qev.Proc.TtyMinor),
					},
					Cwd:      qev.Cwd,
					Argv:     qev.Cmdline,
					Filename: qev.Comm,
				}
				if qev.ExitEvent != nil {
					pr.ExitCode = qev.ExitEvent.ExitCode
				}
				logger.Errorf("MWOLF: Inserting PID %v", pr.PIDs.Tgid)
				p.db.InsertProcess(pr)

				//				if qev.ExitEvent == nil {
				//					pe :=  types.ProcessExecEvent{
				//						PIDs: types.PIDInfo {
				//							Tid: qev.Pid,
				//							Tgid: qev.Pid,
				//							Ppid: qev.Proc.Ppid,
				//							Pgid: qev.Pid,
				//							Sid: qev.Proc.Sid,
				//							StartTimeNS: qev.Proc.TimeBoot,
				//						},
				//						Creds: types.CredInfo{
				//							Ruid: qev.Proc.Uid,
				//							Rgid: qev.Proc.Gid,
				//							Euid: qev.Proc.Euid,
				//							Egid: qev.Proc.Egid,
				//							Suid: qev.Proc.Suid,
				//							Sgid: qev.Proc.Sgid,
				//							CapPermitted: qev.Proc.CapPermitted,
				//							CapEffective: qev.Proc.CapEffective,
				//							},
				//						CWD: qev.Cwd,
				//						Argv: qev.Cmdline,
				//						Filename: qev.Comm,
				//					}
				//					p.db.InsertExec(pe)
				//				} else {
				//					// Exit event
				//					pe := types.ProcessExitEvent {
				//						PIDs: types.PIDInfo {
				//							Tid: qev.Pid,
				//							Tgid: qev.Pid,
				//							Ppid: qev.Proc.Ppid,
				//							Sid: qev.Proc.Sid,
				//							StartTimeNS: qev.Proc.TimeBoot,
				//						},
				//						ExitCode: qev.ExitEvent.ExitCode,
				//					}
				//					p.db.InsertExit(pe)
				//				}
			}
			if len(qevs) == 0 {
				err = qq.Block()
				if err != nil {
					logger.Errorf("quark block: %v", err)
					continue
				}
			}
		}
	}(qq, logger)

	return &p, nil
}

const (
	maxWaitLimit      = 4500 * time.Millisecond // Maximum time SyncDB will wait for process
	combinedWaitLimit = 6 * time.Second         // Multiple SyncDB calls will wait up to this amount within resetDuration
	backoffDuration   = 10 * time.Second        // SyncDB will stop waiting for processes for this time
	resetDuration     = 5 * time.Second         // After this amount of times with no backoffs, the combinedWait will be reset
)

var (
	combinedWait   = 0 * time.Millisecond
	inBackoff      = false
	backoffStart   = time.Now()
	since          = time.Now()
	backoffSkipped = 0
)

func (s prvdr) SyncDB(ev *beat.Event, pid uint32) error {
	if s.db.HasProcess(pid) {
		return nil
	}

	now := time.Now()
	if inBackoff {
		if now.Sub(backoffStart) > backoffDuration {
			s.logger.Warnf("ended backoff, skipped %d processes", backoffSkipped)
			inBackoff = false
			combinedWait = 0 * time.Millisecond
		} else {
			backoffSkipped += 1
			return nil
		}
	} else {
		if combinedWait > combinedWaitLimit {
			s.logger.Warn("starting backoff")
			inBackoff = true
			backoffStart = now
			backoffSkipped = 0
			return nil
		}
		// maintain a moving window of time for the delays we track
		if now.Sub(since) > resetDuration {
			since = now
			combinedWait = 0 * time.Millisecond
		}
	}

	start := now
	nextWait := 5 * time.Millisecond
	for {
		waited := time.Since(start)
		if s.db.HasProcess(pid) {
			s.logger.Debugf("got process that was missing after %v", waited)
			combinedWait = combinedWait + waited
			return nil
		}
		if waited >= maxWaitLimit {
			e := fmt.Errorf("process %v was not seen after %v", pid, waited)
			s.logger.Warnf("%w", e)
			combinedWait = combinedWait + waited
			return e
		}
		time.Sleep(nextWait)
		if nextWait*2+waited > maxWaitLimit {
			nextWait = maxWaitLimit - waited
		} else {
			nextWait = nextWait * 2
		}
	}
}
