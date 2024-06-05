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
	qq     *quark.Queue
}

func NewProvider(ctx context.Context, logger *logp.Logger, db *processdb.DB) (provider.Provider, error) {

	attr := quark.DefaultQueueAttr()
	attr.Flags = quark.QQ_KPROBE
	qq, err := quark.OpenQueue(attr, 64)
	if err != nil {
		return nil, fmt.Errorf("open queue: %v", err)
	}

	p := prvdr{
		ctx:    ctx,
		logger: logger,
		qq:     qq,
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

func (s prvdr) GetProcess(pid uint32) (*types.Process, error) {
	p := s.qq.Lookup(int(pid))
	if p == nil {
		return nil, fmt.Errorf("pid %v not found in quark cache", pid)
	}

	proc := types.Process {
		EntityID: fmt.Sprintf("%s__%s", p.Comm, p.Pid),
		Executable: p.Comm,
		Name: p.Comm,
		Start: starttime(p.Proc.TimeBoot),
		End: endtime(p.ExitEvent.ExitTimeEvent),
		ExitCode: p.ExitEvent.ExitCode,
		Interactive: interactive(p.Proc.TtyMajor, p.Proc.TtyMinor),
		WorkingDirectory: p.Cwd,
		User: struct{
			ID: p.Proc.Euid,
			Name: usernameFromId(p.Proc.Euid),
		},
		Group: struct {
			ID: p.Proc.Eguid,
			Name: groupnameFromId(p.Proc.Eguid),
		},
	}
	return &proc, nil
}

func (s prvdr) SyncDB(ev *beat.Event, pid uint32) error {

	timeout := 5 * time.Second
	ch := time.After(timeout)
	for {
		select {
		case <- ch:
			s.logger.Errorf("%v not seen after %v", pid, timeout)
			break
		default:
			p := s.qq.Lookup(int(pid))
			//TODO: check event, eg. make sure exit seen when enriching exit event
			if p != nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func interactiveFromTTY(tty types.TTYDev) bool {
	return TTYUnknown != getTTYType(tty.Major, tty.Minor)
}

func fullProcessFromDBProcess(p Process) types.Process {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(p.PIDs.StartTimeNS)
	interactive := interactiveFromTTY(p.CTTY)

	ret := types.Process{
		PID:              p.PIDs.Tgid,
		Start:            timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime),
		Name:             basename(p.Filename),
		Executable:       p.Filename,
		Args:             p.Argv,
		WorkingDirectory: p.Cwd,
		Interactive:      &interactive,
	}

	euid := p.Creds.Euid
	egid := p.Creds.Egid
	ret.User.ID = strconv.FormatUint(uint64(euid), 10)
	username, ok := getUserName(ret.User.ID)
	if ok {
		ret.User.Name = username
	}
	ret.Group.ID = strconv.FormatUint(uint64(egid), 10)
	groupname, ok := getGroupName(ret.Group.ID)
	if ok {
		ret.Group.Name = groupname
	}
	ret.Thread.Capabilities.Permitted, _ = capabilities.FromUint64(p.Creds.CapPermitted)
	ret.Thread.Capabilities.Effective, _ = capabilities.FromUint64(p.Creds.CapEffective)
	ret.TTY.CharDevice.Major = p.CTTY.Major
	ret.TTY.CharDevice.Minor = p.CTTY.Minor
	ret.ExitCode = p.ExitCode

	return ret
}

func fillParent(process *types.Process, parent Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(parent.PIDs.StartTimeNS)

	interactive := interactiveFromTTY(parent.CTTY)
	euid := parent.Creds.Euid
	egid := parent.Creds.Egid
	process.Parent.PID = parent.PIDs.Tgid
	process.Parent.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.Parent.Name = basename(parent.Filename)
	process.Parent.Executable = parent.Filename
	process.Parent.Args = parent.Argv
	process.Parent.WorkingDirectory = parent.Cwd
	process.Parent.Interactive = &interactive
	process.Parent.User.ID = strconv.FormatUint(uint64(euid), 10)
	username, ok := getUserName(process.Parent.User.ID)
	if ok {
		process.Parent.User.Name = username
	}
	process.Parent.Group.ID = strconv.FormatUint(uint64(egid), 10)
	groupname, ok := getGroupName(process.Parent.Group.ID)
	if ok {
		process.Parent.Group.Name = groupname
	}
}

func fillGroupLeader(process *types.Process, groupLeader Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(groupLeader.PIDs.StartTimeNS)

	interactive := interactiveFromTTY(groupLeader.CTTY)
	euid := groupLeader.Creds.Euid
	egid := groupLeader.Creds.Egid
	process.GroupLeader.PID = groupLeader.PIDs.Tgid
	process.GroupLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.GroupLeader.Name = basename(groupLeader.Filename)
	process.GroupLeader.Executable = groupLeader.Filename
	process.GroupLeader.Args = groupLeader.Argv
	process.GroupLeader.WorkingDirectory = groupLeader.Cwd
	process.GroupLeader.Interactive = &interactive
	process.GroupLeader.User.ID = strconv.FormatUint(uint64(euid), 10)
	username, ok := getUserName(process.GroupLeader.User.ID)
	if ok {
		process.GroupLeader.User.Name = username
	}
	process.GroupLeader.Group.ID = strconv.FormatUint(uint64(egid), 10)
	groupname, ok := getGroupName(process.GroupLeader.Group.ID)
	if ok {
		process.GroupLeader.Group.Name = groupname
	}
}

func fillSessionLeader(process *types.Process, sessionLeader Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(sessionLeader.PIDs.StartTimeNS)

	interactive := interactiveFromTTY(sessionLeader.CTTY)
	euid := sessionLeader.Creds.Euid
	egid := sessionLeader.Creds.Egid
	process.SessionLeader.PID = sessionLeader.PIDs.Tgid
	process.SessionLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.SessionLeader.Name = basename(sessionLeader.Filename)
	process.SessionLeader.Executable = sessionLeader.Filename
	process.SessionLeader.Args = sessionLeader.Argv
	process.SessionLeader.WorkingDirectory = sessionLeader.Cwd
	process.SessionLeader.Interactive = &interactive
	process.SessionLeader.User.ID = strconv.FormatUint(uint64(euid), 10)
	username, ok := getUserName(process.SessionLeader.User.ID)
	if ok {
		process.SessionLeader.User.Name = username
	}
	process.SessionLeader.Group.ID = strconv.FormatUint(uint64(egid), 10)
	groupname, ok := getGroupName(process.SessionLeader.Group.ID)
	if ok {
		process.SessionLeader.Group.Name = groupname
	}
}

func fillEntryLeader(process *types.Process, entryType EntryType, entryLeader Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(entryLeader.PIDs.StartTimeNS)

	interactive := interactiveFromTTY(entryLeader.CTTY)
	euid := entryLeader.Creds.Euid
	egid := entryLeader.Creds.Egid
	process.EntryLeader.PID = entryLeader.PIDs.Tgid
	process.EntryLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.EntryLeader.Name = basename(entryLeader.Filename)
	process.EntryLeader.Executable = entryLeader.Filename
	process.EntryLeader.Args = entryLeader.Argv
	process.EntryLeader.WorkingDirectory = entryLeader.Cwd
	process.EntryLeader.Interactive = &interactive
	process.EntryLeader.User.ID = strconv.FormatUint(uint64(euid), 10)
	username, ok := getUserName(process.EntryLeader.User.ID)
	if ok {
		process.EntryLeader.User.Name = username
	}
	process.EntryLeader.Group.ID = strconv.FormatUint(uint64(egid), 10)
	groupname, ok := getGroupName(process.EntryLeader.Group.ID)
	if ok {
		process.EntryLeader.Group.Name = groupname
	}

	process.EntryLeader.EntryMeta.Type = string(entryType)
}


