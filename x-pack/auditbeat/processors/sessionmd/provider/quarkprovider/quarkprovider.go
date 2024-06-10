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
	attr.Flags = quark.QQ_KPROBE | quark.QQ_MIN_AGG
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
				logger.Debugf("qev: %v", qev)
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

func (p prvdr) SyncDB(ev *beat.Event, pid uint32) error {
	time.Sleep(100 * time.Millisecond)
	return nil

	// // TODO: Not working correctly...
	//
	//	timeout := 5 * time.Second
	//	ch := time.After(timeout)
	//	for {
	//		select {
	//		case <- ch:
	//			p.logger.Errorf("%v not seen after %v", pid, timeout)
	//			break
	//		default:
	//			proc := p.qq.Lookup(int(pid))
	//			//TODO: check event, eg. make sure exit seen when enriching exit event
	//			if proc != nil {
	//				break
	//			}
	//			time.Sleep(200 * time.Millisecond)
	//		}
	//	}
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
		PID:        qev.Pid,
		Start:      timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime),
		Name:       basename(qev.Filename),
		Executable: qev.Filename,
		//		Args:             qev.Argv,
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
	p.fillGroupLeader(&ret, qev.Proc.Pgid)
	p.fillSessionLeader(&ret, qev.Proc.Sid)
	p.fillEntryLeader(&ret, Init, uint32(1))
	setEntityID(&ret)
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
	//	process.Parent.Args = qev.Argv
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
	//	process.GroupLeader.Args = qev.Argv
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
	//	process.SessionLeader.Args = qev.Argv
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

func (p prvdr) fillEntryLeader(process *types.Process, entryType EntryType, elid uint32) {
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
	//	process.EntryLeader.Args = qev.Argv
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
	process.EntryLeader.EntryMeta.Type = string(entryType)
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
