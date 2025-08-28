// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (amd64 || arm64) && cgo

package kerneltracingprovider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	quark "github.com/elastic/go-quark"

	"github.com/elastic/beats/v7/auditbeat/helper/tty"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/provider"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type prvdr struct {
	ctx            context.Context
	logger         *logp.Logger
	qq             *quark.Queue
	qqMtx          *sync.Mutex
	combinedWait   time.Duration
	inBackoff      bool
	backoffStart   time.Time
	since          time.Time
	backoffSkipped int
}

const (
	Init         = "init"
	Sshd         = "sshd"
	Ssm          = "ssm"
	Container    = "container"
	Terminal     = "terminal"
	Kthread      = "kthread"
	EntryConsole = "console"
	EntryUnknown = "unknown"
)

var (
	bootID     string
	pidNsInode uint64
)

// readBootID returns the boot ID of the Linux system from "/proc/sys/kernel/random/boot_id"
func readBootID() (string, error) {
	bootID, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", fmt.Errorf("could not read /proc/sys/kernel/random/boot_id: %w", err)
	}

	return strings.TrimRight(string(bootID), "\n"), nil
}

// readPIDNsInode returns the PID namespace inode that auditbeat is running in from "/proc/self/ns/pid"
func readPIDNsInode() (uint64, error) {
	var ret uint64

	pidNsInodeRaw, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		return 0, fmt.Errorf("could not read /proc/self/ns/pid: %w", err)
	}

	if _, err = fmt.Sscanf(pidNsInodeRaw, "pid:[%d]", &ret); err != nil {
		return 0, fmt.Errorf("could not parse contents of /proc/self/ns/pid (%q): %w", pidNsInodeRaw, err)
	}

	return ret, nil
}

// NewProvider returns a new instance of kerneltracingprovider
func NewProvider(ctx context.Context, logger *logp.Logger, reg *monitoring.Registry) (provider.Provider, error) {
	attr := quark.DefaultQueueAttr()
	attr.Flags = quark.QQ_ALL_BACKENDS | quark.QQ_ENTRY_LEADER | quark.QQ_NO_SNAPSHOT
	qq, err := quark.OpenQueue(attr, 64)
	if err != nil {
		return nil, fmt.Errorf("open queue: %w", err)
	}

	procMetrics := NewStats(reg)

	p := &prvdr{
		ctx:            ctx,
		logger:         logger,
		qq:             qq,
		qqMtx:          new(sync.Mutex),
		combinedWait:   0 * time.Millisecond,
		inBackoff:      false,
		backoffStart:   time.Now(),
		since:          time.Now(),
		backoffSkipped: 0,
	}

	go func(ctx context.Context, qq *quark.Queue, logger *logp.Logger, p *prvdr, stats *Stats) {

		lastUpdate := time.Now()

		defer qq.Close()
		for ctx.Err() == nil {
			p.qqMtx.Lock()
			events, err := qq.GetEvents()
			p.qqMtx.Unlock()
			if err != nil {
				logger.Errorw("get events from quark, no more process enrichment from this processor will be done", "error", err)
				break
			}
			if time.Since(lastUpdate) > time.Second*5 {
				p.qqMtx.Lock()
				metrics := qq.Stats()
				p.qqMtx.Unlock()

				stats.Aggregations.Set(metrics.Aggregations)
				stats.Insertions.Set(metrics.Insertions)
				stats.Lost.Set(metrics.Lost)
				stats.NonAggregations.Set(metrics.NonAggregations)
				stats.Removals.Set(metrics.Removals)
				lastUpdate = time.Now()
			}

			if len(events) == 0 {
				err = qq.Block()
				if err != nil {
					logger.Errorw("quark block, no more process enrichment from this processor will be done", "error", err)
					break
				}
			}
		}
	}(ctx, qq, logger, p, procMetrics)

	bootID, err = readBootID()
	if err != nil {
		p.logger.Errorw("failed to read boot ID, entity ID will not be correct", "error", err)
	}
	pidNsInode, err = readPIDNsInode()
	if err != nil {
		p.logger.Errorw("failed to read PID namespace inode, entity ID will not be correct", "error", err)
	}

	return p, nil
}

const (
	maxWaitLimit      = 1200 * time.Millisecond // Maximum time SyncDB will wait for process
	combinedWaitLimit = 15 * time.Second        // Multiple SyncDB calls will wait up to this amount within resetDuration
	backoffDuration   = 10 * time.Second        // SyncDB will stop waiting for processes for this time
	resetDuration     = 5 * time.Second         // After this amount of times with no backoffs, the combinedWait will be reset
)

// Sync ensures that the specified pid is present in the internal cache, to ensure the processor is capable of enriching the process.
// The function waits up to a maximum limit (maxWaitLimit) for the pid to appear in the cache using an exponential delay strategy.
// If the pid is not found within the time limit, then an error is returned.
//
// The function also maintains a moving window of time for tracking delays, and applies a backoff strategy if the combined wait time
// exceeds a certain limit (combinedWaitLimit). This is done so that in the case where there are multiple delays, the cumulative delay
// does not exceed a reasonable threshold that would delay all other events processed by auditbeat. When in the backoff state, enrichment
// will proceed without waiting for the process data to exist in the cache, likely resulting in missing enrichment data.
func (p *prvdr) Sync(_ *beat.Event, pid uint32) error {
	// If pid is already in qq, return immediately
	if _, found := p.lookupLocked(pid); found {
		return nil
	}

	start := time.Now()

	p.handleBackoff(start)
	if p.inBackoff {
		return nil
	}

	// Wait until either the process exists within the cache or the maxWaitLimit is exceeded, with an exponential delay
	nextWait := 5 * time.Millisecond
	for {
		waited := time.Since(start)
		if _, found := p.lookupLocked(pid); found {
			p.logger.Debugw("got process that was missing ", "waited", waited)
			p.combinedWait = p.combinedWait + waited
			return nil
		}
		if waited >= maxWaitLimit {
			p.combinedWait = p.combinedWait + waited
			return fmt.Errorf("process %v was not seen after %v", pid, waited)
		}
		time.Sleep(nextWait)
		if nextWait*2+waited > maxWaitLimit {
			nextWait = maxWaitLimit - waited
		} else {
			nextWait = nextWait * 2
		}
	}
}

// handleBackoff handles backoff logic of `Sync`
// If the combinedWait time exceeds the combinedWaitLimit duration, the provider will go into backoff state until the backoffDuration is exceeded.
// If in a backoff period, it will track the number of skipped processes, and then log the number when exiting backoff.
//
// If there have been no backoffs within the resetDuration, the combinedWait duration is reset to zero, to keep a moving window in which delays are tracked.
func (p *prvdr) handleBackoff(now time.Time) {
	if p.inBackoff {
		if now.Sub(p.backoffStart) > backoffDuration {
			p.logger.Infow("ended backoff, skipped processes", "backoffSkipped", p.backoffSkipped)
			p.inBackoff = false
			p.combinedWait = 0 * time.Millisecond
		} else {
			p.backoffSkipped += 1
			return
		}
	} else {
		if p.combinedWait > combinedWaitLimit {
			p.logger.Info("starting backoff")
			p.inBackoff = true
			p.backoffStart = now
			p.backoffSkipped = 0
			return
		}
		if now.Sub(p.since) > resetDuration {
			p.since = now
			p.combinedWait = 0 * time.Millisecond
		}
	}
}

// GetProcess returns a reference to Process struct that contains all known information for the
// process, and its ancestors (parent, process group leader, session leader, and entry leader).
func (p *prvdr) GetProcess(pid uint32) (*types.Process, error) {
	proc, found := p.lookupLocked(pid)
	if !found {
		return nil, fmt.Errorf("PID %d not found in cache", pid)
	}

	interactive := tty.InteractiveFromTTY(tty.TTYDev{
		Major: proc.Proc.TtyMajor,
		Minor: proc.Proc.TtyMinor,
	})

	start := time.Unix(0, int64(proc.Proc.TimeBoot))

	ret := types.Process{
		PID:              proc.Pid,
		Start:            &start,
		Name:             basename(proc.Filename),
		Executable:       proc.Filename,
		Args:             proc.Cmdline,
		WorkingDirectory: proc.Cwd,
		Interactive:      &interactive,
	}

	euid := proc.Proc.Euid
	egid := proc.Proc.Egid
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
	ret.TTY.CharDevice.Major = uint16(proc.Proc.TtyMajor)
	ret.TTY.CharDevice.Minor = uint16(proc.Proc.TtyMinor)
	if proc.Exit.Valid {
		end := time.Unix(0, int64(proc.Exit.ExitTimeProcess))
		ret.ExitCode = proc.Exit.ExitCode
		ret.End = &end
	}
	ret.EntityID = calculateEntityIDv1(pid, *ret.Start)

	p.fillParent(&ret, proc.Proc.Ppid)
	p.fillGroupLeader(&ret, proc.Proc.Pgid)
	p.fillSessionLeader(&ret, proc.Proc.Sid)
	p.fillEntryLeader(&ret, proc.Proc.EntryLeader)
	setEntityID(&ret)
	setSameAsProcess(&ret)
	return &ret, nil
}

func (p prvdr) lookupLocked(pid uint32) (quark.Process, bool) {
	p.qqMtx.Lock()
	defer p.qqMtx.Unlock()

	return p.qq.Lookup(int(pid))
}

// fillParent populates the parent process fields with the attributes of the process with PID `ppid`
func (p prvdr) fillParent(process *types.Process, ppid uint32) {
	proc, found := p.lookupLocked(ppid)
	if !found {
		return
	}

	start := time.Unix(0, int64(proc.Proc.TimeBoot))
	interactive := tty.InteractiveFromTTY(tty.TTYDev{
		Major: proc.Proc.TtyMajor,
		Minor: proc.Proc.TtyMinor,
	})
	euid := proc.Proc.Euid
	egid := proc.Proc.Egid
	process.Parent.PID = proc.Pid
	process.Parent.Start = &start
	process.Parent.Name = basename(proc.Filename)
	process.Parent.Executable = proc.Filename
	process.Parent.Args = proc.Cmdline
	process.Parent.WorkingDirectory = proc.Cwd
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

// fillGroupLeader populates the process group leader fields with the attributes of the process with PID `pgid`
func (p prvdr) fillGroupLeader(process *types.Process, pgid uint32) {
	proc, found := p.lookupLocked(pgid)
	if !found {
		return
	}

	start := time.Unix(0, int64(proc.Proc.TimeBoot))

	interactive := tty.InteractiveFromTTY(tty.TTYDev{
		Major: proc.Proc.TtyMajor,
		Minor: proc.Proc.TtyMinor,
	})
	euid := proc.Proc.Euid
	egid := proc.Proc.Egid
	process.GroupLeader.PID = proc.Pid
	process.GroupLeader.Start = &start
	process.GroupLeader.Name = basename(proc.Filename)
	process.GroupLeader.Executable = proc.Filename
	process.GroupLeader.Args = proc.Cmdline
	process.GroupLeader.WorkingDirectory = proc.Cwd
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

// fillSessionLeader populates the session leader fields with the attributes of the process with PID `sid`
func (p prvdr) fillSessionLeader(process *types.Process, sid uint32) {
	proc, found := p.lookupLocked(sid)
	if !found {
		return
	}

	start := time.Unix(0, int64(proc.Proc.TimeBoot))

	interactive := tty.InteractiveFromTTY(tty.TTYDev{
		Major: proc.Proc.TtyMajor,
		Minor: proc.Proc.TtyMinor,
	})
	euid := proc.Proc.Euid
	egid := proc.Proc.Egid
	process.SessionLeader.PID = proc.Pid
	process.SessionLeader.Start = &start
	process.SessionLeader.Name = basename(proc.Filename)
	process.SessionLeader.Executable = proc.Filename
	process.SessionLeader.Args = proc.Cmdline
	process.SessionLeader.WorkingDirectory = proc.Cwd
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

// fillEntryLeader populates the entry leader fields with the attributes of the process with PID `elid`
func (p prvdr) fillEntryLeader(process *types.Process, elid uint32) {
	proc, found := p.lookupLocked(elid)
	if !found {
		return
	}

	start := time.Unix(0, int64(proc.Proc.TimeBoot))

	interactive := tty.InteractiveFromTTY(tty.TTYDev{
		Major: proc.Proc.TtyMajor,
		Minor: proc.Proc.TtyMinor,
	})

	euid := proc.Proc.Euid
	egid := proc.Proc.Egid
	process.EntryLeader.PID = proc.Pid
	process.EntryLeader.Start = &start
	process.EntryLeader.WorkingDirectory = proc.Cwd
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
	process.EntryLeader.EntryMeta.Type = getEntryTypeName(proc.Proc.EntryLeaderType)
}

// setEntityID sets entityID for the process and its parent, group leader, session leader, entry leader if possible
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

// setSameAsProcess sets if the process is the same as its group leader, session leader, entry leader
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

// calculateEntityIDv1 calculates the entity ID for a process.
// This is a globally unique identifier for the process.
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

	return filepath.Base(pathStr)
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

func getEntryTypeName(entryType uint32) string {
	switch int(entryType) {
	case quark.QUARK_ELT_INIT:
		return Init
	case quark.QUARK_ELT_SSHD:
		return Sshd
	case quark.QUARK_ELT_SSM:
		return Ssm
	case quark.QUARK_ELT_CONTAINER:
		return Container
	case quark.QUARK_ELT_TERM:
		return Terminal
	case quark.QUARK_ELT_CONSOLE:
		return EntryConsole
	case quark.QUARK_ELT_KTHREAD:
		return Kthread
	default:
		return "unknown"
	}
}
