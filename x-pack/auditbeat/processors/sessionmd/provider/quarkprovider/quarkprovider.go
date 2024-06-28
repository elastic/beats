// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package quarkprovider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/timeutils"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	quark "github.com/elastic/quark/go"
)

type prvdr struct {
	ctx    context.Context
	logger *logp.Logger
	qq     *quark.Queue
}

type TTYType int

const (
	TTYUnknown TTYType = iota
	Pts
	TTY
	TTYConsole
)

type EntryType string

const (
	Init         EntryType = "init"
	Sshd         EntryType = "sshd"
	Ssm          EntryType = "ssm"
	Container    EntryType = "container"
	Terminal     EntryType = "terminal"
	EntryConsole EntryType = "console"
	EntryUnknown EntryType = "unknown"
)

var containerRuntimes = [...]string{
	"containerd-shim",
	"runc",
	"conmon",
}

// "filtered" executables are executables that relate to internal
// implementation details of entry mechanisms. The set of circumstances under
// which they can become an entry leader are reduced compared to other binaries
// (see implementation and unit tests).
var filteredExecutables = [...]string{
	"runc",
	"containerd-shim",
	"calico-node",
	"check-status",
	"conmon",
}

const (
	ptsMinMajor     = 136
	ptsMaxMajor     = 143
	ttyMajor        = 4
	consoleMaxMinor = 63
	ttyMaxMinor     = 255
	retryCount      = 2
)

var (
	bootID     string
	pidNsInode uint64
	initError  error
	once       sync.Once
)

func readBootID() (string, error) {
	bootID, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		panic(fmt.Sprintf("could not read /proc/sys/kernel/random/boot_id: %v", err))
	}

	return strings.TrimRight(string(bootID), "\n"), nil
}

func readPIDNsInode() (uint64, error) {
	var ret uint64

	pidNsInodeRaw, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		panic(fmt.Sprintf("could not read /proc/self/ns/pid: %v", err))
	}

	if _, err = fmt.Sscanf(pidNsInodeRaw, "pid:[%d]", &ret); err != nil {
		panic(fmt.Sprintf("could not parse contents of /proc/self/ns/pid (%s): %v", pidNsInodeRaw, err))
	}

	return ret, nil
}

func NewProvider(ctx context.Context, logger *logp.Logger) (provider.Provider, error) {

	attr := quark.DefaultQueueAttr()
	attr.Flags = quark.QQ_KPROBE | quark.QQ_MIN_AGG | quark.QQ_ENTRY_LEADER
	qq, err := quark.OpenQueue(attr, 64)
	if err != nil {
		return nil, fmt.Errorf("open queue: %v", err)
	}

	p := prvdr{
		ctx:    ctx,
		logger: logger,
		qq:     qq,
	}

	go func(qq *quark.Queue, logger *logp.Logger) {
		for {
			qevs, err := qq.GetEvents()
			if err != nil {
				logger.Errorf("get events from quark: %v", err)
				continue
			}
			for _, qev := range qevs {
				logger.Infof("qev: %v", qev)
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

	bootID, _ = readBootID()
	pidNsInode, _ = readPIDNsInode()

	return &p, nil
}

const (
	maxWaitLimit      = 1200 * time.Millisecond // Maximum time SyncDB will wait for process
	combinedWaitLimit = 15 * time.Second        // Multiple SyncDB calls will wait up to this amount within resetDuration
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

func (p prvdr) SyncDB(ev *beat.Event, pid uint32) error {
	if qev := p.qq.Lookup(int(pid)); qev != nil {
		return nil
	}

	now := time.Now()
	if inBackoff {
		if now.Sub(backoffStart) > backoffDuration {
			p.logger.Warnf("ended backoff, skipped %d processes", backoffSkipped)
			inBackoff = false
			combinedWait = 0 * time.Millisecond
		} else {
			backoffSkipped += 1
			return nil
		}
	} else {
		if combinedWait > combinedWaitLimit {
			p.logger.Warn("starting backoff")
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
		if qev := p.qq.Lookup(int(pid)); qev != nil {
			p.logger.Debugf("got process that was missing after %v", waited)
			combinedWait = combinedWait + waited
			return nil
		}
		if waited >= maxWaitLimit {
			e := fmt.Errorf("process %v was not seen after %v", pid, waited)
			p.logger.Warnf("%w", e)
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

func (p prvdr) GetProcess(pid uint32) (*types.Process, error) {
	qev := p.qq.Lookup(int(pid))
	if qev == nil {
		return nil, fmt.Errorf("PID %d not found in cache", pid)
	}

	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(qev.Proc.TimeBoot)
	interactive := interactiveFromTTY(types.TTYDev{
		Major: qev.Proc.TtyMajor,
		Minor: qev.Proc.TtyMinor,
	})

	ret := types.Process{
		PID:              qev.Pid,
		Start:            timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime),
		Name:             basename(qev.Filename),
		Executable:       qev.Filename,
		Args:             []string{qev.Filename}, // TODO: Fix
		WorkingDirectory: qev.Cwd,
		Interactive:      &interactive,
	}

	euid := qev.Proc.Euid
	egid := qev.Proc.Egid
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
	ret.TTY.CharDevice.Major = uint16(qev.Proc.TtyMajor)
	ret.TTY.CharDevice.Minor = uint16(qev.Proc.TtyMinor)
	if qev.ExitEvent != nil {
		ret.ExitCode = qev.ExitEvent.ExitCode
	}
	ret.EntityID = calculateEntityIDv1(pid, *ret.Start)

	p.fillParent(&ret, qev.Proc.Ppid)
	p.fillGroupLeader(&ret, qev.Pid) //  qev.Proc.Pgid)
	p.fillSessionLeader(&ret, qev.Proc.Sid)
	p.fillEntryLeader(&ret, qev.Proc.EntryLeaderType, qev.Proc.EntryLeader)
	setEntityID(&ret)
	setSameAsProcess(&ret)
	return &ret, nil
}

func (p prvdr) fillParent(process *types.Process, ppid uint32) {
	qev := p.qq.Lookup(int(ppid))
	if qev == nil {
		return
	}

	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(qev.Proc.TimeBoot)
	interactive := interactiveFromTTY(types.TTYDev{
		Major: qev.Proc.TtyMajor,
		Minor: qev.Proc.TtyMinor,
	})
	euid := qev.Proc.Euid
	egid := qev.Proc.Egid
	process.Parent.PID = qev.Proc.Ppid
	process.Parent.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.Parent.Name = basename(qev.Filename)
	process.Parent.Executable = qev.Filename
	process.Parent.Args = []string{qev.Filename} //TODO: FIx
	process.Parent.WorkingDirectory = qev.Cwd
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
	process.Parent.EntityID = calculateEntityIDv1(ppid, *process.Start)
}

func (p prvdr) fillGroupLeader(process *types.Process, pgid uint32) {
	qev := p.qq.Lookup(int(pgid))
	if qev == nil {
		return
	}

	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(qev.Proc.TimeBoot)

	interactive := interactiveFromTTY(types.TTYDev{
		Major: qev.Proc.TtyMajor,
		Minor: qev.Proc.TtyMinor,
	})
	euid := qev.Proc.Euid
	egid := qev.Proc.Egid
	process.GroupLeader.PID = qev.Pid
	process.GroupLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.GroupLeader.Name = basename(qev.Filename)
	process.GroupLeader.Executable = qev.Filename
	process.GroupLeader.Args = []string{qev.Filename} //TODO: fix
	process.GroupLeader.WorkingDirectory = qev.Cwd
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
	process.GroupLeader.EntityID = calculateEntityIDv1(pgid, *process.GroupLeader.Start)
}

func (p prvdr) fillSessionLeader(process *types.Process, sid uint32) {
	qev := p.qq.Lookup(int(sid))
	if qev == nil {
		return
	}

	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(qev.Proc.TimeBoot)

	interactive := interactiveFromTTY(types.TTYDev{
		Major: qev.Proc.TtyMajor,
		Minor: qev.Proc.TtyMinor,
	})
	euid := qev.Proc.Euid
	egid := qev.Proc.Egid
	process.SessionLeader.PID = qev.Pid
	process.SessionLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.SessionLeader.Name = basename(qev.Filename)
	process.SessionLeader.Executable = qev.Filename
	process.SessionLeader.Args = []string{qev.Filename} //TODO: fix
	process.SessionLeader.WorkingDirectory = qev.Cwd
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
	process.SessionLeader.EntityID = calculateEntityIDv1(sid, *process.SessionLeader.Start)
}

func (p prvdr) fillEntryLeader(process *types.Process, entryType uint32, elid uint32) {
	qev := p.qq.Lookup(int(elid))
	if qev == nil {
		return
	}

	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(qev.Proc.TimeBoot)

	interactive := interactiveFromTTY(types.TTYDev{
		Major: qev.Proc.TtyMajor,
		Minor: qev.Proc.TtyMinor,
	})

	euid := qev.Proc.Euid
	egid := qev.Proc.Egid
	process.EntryLeader.PID = qev.Pid
	process.EntryLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.EntryLeader.Name = basename(qev.Filename)
	process.EntryLeader.Executable = qev.Filename
	process.EntryLeader.Args = []string{qev.Filename} // TODO: Fix
	process.EntryLeader.WorkingDirectory = qev.Cwd
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

	process.EntryLeader.EntityID = calculateEntityIDv1(elid, *process.EntryLeader.Start)
	process.EntryLeader.EntryMeta.Type = "UNKNOWN" //TODO: use real value
}

func setEntityID(process *types.Process) {
	if process.PID != 0 && process.Start != nil {
		process.EntityID = calculateEntityIDv1(process.PID, *process.Start)
	}

	if process.Parent.PID != 0 && process.Parent.Start != nil {
		process.Parent.EntityID = calculateEntityIDv1(process.Parent.PID, *process.Parent.Start)
	}

	if process.GroupLeader.PID != 0 && process.GroupLeader.Start != nil {
		process.GroupLeader.EntityID = calculateEntityIDv1(process.GroupLeader.PID, *process.GroupLeader.Start)
	}

	if process.SessionLeader.PID != 0 && process.SessionLeader.Start != nil {
		process.SessionLeader.EntityID = calculateEntityIDv1(process.SessionLeader.PID, *process.SessionLeader.Start)
	}

	if process.EntryLeader.PID != 0 && process.EntryLeader.Start != nil {
		process.EntryLeader.EntityID = calculateEntityIDv1(process.EntryLeader.PID, *process.EntryLeader.Start)
	}
}

func setSameAsProcess(process *types.Process) {
	if process.GroupLeader.PID != 0 && process.GroupLeader.Start != nil {
		sameAsProcess := process.PID == process.GroupLeader.PID
		process.GroupLeader.SameAsProcess = &sameAsProcess
	}

	if process.SessionLeader.PID != 0 && process.SessionLeader.Start != nil {
		sameAsProcess := process.PID == process.SessionLeader.PID
		process.SessionLeader.SameAsProcess = &sameAsProcess
	}

	if process.EntryLeader.PID != 0 && process.EntryLeader.Start != nil {
		sameAsProcess := process.PID == process.EntryLeader.PID
		process.EntryLeader.SameAsProcess = &sameAsProcess
	}
}

func interactiveFromTTY(tty types.TTYDev) bool {
	return TTYUnknown != getTTYType(tty.Major, tty.Minor)
}

func getTTYType(major uint32, minor uint32) TTYType {
	if major >= ptsMinMajor && major <= ptsMaxMajor {
		return Pts
	}

	if ttyMajor == major {
		if minor <= consoleMaxMinor {
			return TTYConsole
		} else if minor > consoleMaxMinor && minor <= ttyMaxMinor {
			return TTY
		}
	}

	return TTYUnknown
}

func calculateEntityIDv1(pid uint32, startTime time.Time) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(
			fmt.Sprintf("%d__%s__%d__%d",
				pidNsInode,
				bootID,
				uint64(pid),
				uint64(startTime.Unix()),
			),
		),
	)
}

// `path.Base` returns a '.' for empty strings, this just special cases that
// situation to return an empty string
func basename(pathStr string) string {
	if pathStr == "" {
		return ""
	}

	return path.Base(pathStr)
}

// getUserName will return the name associated with the user ID, if it exists
func getUserName(id string) (string, bool) {
	user, err := user.LookupId(id)
	if err != nil {
		return "", false
	}
	return user.Username, true
}

// getGroupName will return the name associated with the group ID, if it exists
func getGroupName(id string) (string, bool) {
	group, err := user.LookupGroupId(id)
	if err != nil {
		return "", false
	}
	return group.Name, true
}
