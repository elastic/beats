// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/auditbeat/helper/tty"
	"github.com/elastic/beats/v7/libbeat/common/capabilities"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/timeutils"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
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
	retryCount = 2
)

type Process struct {
	PIDs     types.PIDInfo
	Creds    types.CredInfo
	CTTY     tty.TTYDev
	Argv     []string
	Cwd      string
	Filename string
	ExitCode int32

	// procfsLookupFail is true if procfs couldn't find a matching PID in /proc.
	procfsLookupFail bool
	insertTime       time.Time
}

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

func pidInfoFromProto(p types.PIDInfo) types.PIDInfo {
	return types.PIDInfo{
		StartTimeNS: p.StartTimeNS,
		Tid:         p.Tid,
		Tgid:        p.Tgid,
		Vpid:        p.Vpid,
		Ppid:        p.Ppid,
		Pgid:        p.Pgid,
		Sid:         p.Sid,
	}
}

func credInfoFromProto(p types.CredInfo) types.CredInfo {
	return types.CredInfo{
		Ruid:         p.Ruid,
		Rgid:         p.Rgid,
		Euid:         p.Euid,
		Egid:         p.Egid,
		Suid:         p.Suid,
		Sgid:         p.Sgid,
		CapPermitted: p.CapPermitted,
		CapEffective: p.CapEffective,
	}
}

func ttyTermiosFromProto(p tty.TTYTermios) tty.TTYTermios {
	return tty.TTYTermios{
		CIflag: p.CIflag,
		COflag: p.COflag,
		CLflag: p.CLflag,
		CCflag: p.CCflag,
	}
}

func ttyWinsizeFromProto(p tty.TTYWinsize) tty.TTYWinsize {
	return tty.TTYWinsize{
		Rows: p.Rows,
		Cols: p.Cols,
	}
}

func ttyDevFromProto(p tty.TTYDev) tty.TTYDev {
	return tty.TTYDev{
		Major:   p.Major,
		Minor:   p.Minor,
		Winsize: ttyWinsizeFromProto(p.Winsize),
		Termios: ttyTermiosFromProto(p.Termios),
	}
}

func initialize() {
	var err error
	bootID, err = readBootID()
	if err != nil {
		initError = err
		return
	}
	pidNsInode, err = readPIDNsInode()
	if err != nil {
		initError = err
	}
}

type DB struct {
	mutex                    sync.RWMutex
	logger                   *logp.Logger
	processes                map[uint32]Process
	entryLeaders             map[uint32]EntryType
	entryLeaderRelationships map[uint32]uint32
	procfs                   procfs.Reader
	stopChan                 chan struct{}
	// map of processes that we can remove during the next reaper run, if the exit event is older than `removalCandidateTimeout`
	removalMap map[uint32]removalCandidate
	ctx        context.Context

	// used for metrics reporting
	stats *Stats

	// knobs for the reaper thread follows

	// how often the reaper checks for expired or orphaned events.
	// A negative value disables the reaper.
	reaperPeriod time.Duration
	// Tells the reaper to remove orphaned process exec events.
	// If true, exec events for which no /proc entry can be found will be removed after their insertion time has passed `orphanTimeout`.
	// If disabled, the reaper will only remove exec events if they are matched with a exit event.
	reapProcesses bool
	// The duration after which we'll reap an orphaned process exec event for which no /proc data exists. Measured from the time the event is inserted.
	processReapAfter time.Duration
}

// NewDB creates a new DB for tracking processes.
//
//   - metrics: monitoring registry for exporting DB metrics
//   - reader: handler for /proc data and events.
//   - reaperPeriod: tells the reaper to update its tracking of deat and orphaned processes every at every `n` period.
//   - reapProcesses: optionally tell the reaper to also reap orphan processes from the DB, if no matching exit event can be found.
//     May result in data loss if the DB in under load and events do not arrive in a timely fashion.
func NewDB(ctx context.Context, metrics *monitoring.Registry, reader procfs.Reader, logger *logp.Logger, reaperPeriod time.Duration, reapProcesses bool) (*DB, error) {
	once.Do(initialize)
	if initError != nil {
		return &DB{}, initError
	}
	db := DB{
		logger:                   logger.Named("processdb"),
		processes:                make(map[uint32]Process),
		entryLeaders:             make(map[uint32]EntryType),
		entryLeaderRelationships: make(map[uint32]uint32),
		procfs:                   reader,
		stopChan:                 make(chan struct{}),
		removalMap:               make(map[uint32]removalCandidate),
		reaperPeriod:             reaperPeriod,
		stats:                    NewStats(metrics),
		reapProcesses:            reapProcesses,
		processReapAfter:         time.Minute * 10,
		ctx:                      ctx,
	}

	if db.reaperPeriod > 0 {
		logger.Infof("starting processDB reaper with interval %s", db.reaperPeriod)
	}

	if db.reapProcesses {
		logger.Info("WARNING: reaping orphaned processes. May result in data loss.")
	}
	db.startReaper()
	return &db, nil
}

func (db *DB) calculateEntityIDv1(pid uint32, startTime time.Time) string {
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

func (db *DB) InsertFork(fork types.ProcessForkEvent) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	pid := fork.ChildPIDs.Tgid
	ppid := fork.ParentPIDs.Tgid

	if entry, ok := db.processes[ppid]; ok {
		entry.PIDs = pidInfoFromProto(fork.ChildPIDs)
		entry.Creds = credInfoFromProto(fork.Creds)
		db.processes[pid] = entry
		if entryPID, ok := db.entryLeaderRelationships[ppid]; ok {
			db.entryLeaderRelationships[pid] = entryPID
		}
	} else {
		db.processes[pid] = Process{
			PIDs:  pidInfoFromProto(fork.ChildPIDs),
			Creds: credInfoFromProto(fork.Creds),
		}
	}
}

func (db *DB) InsertProcess(process Process) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.insertProcess(process)
}

func (db *DB) insertProcess(process Process) {
	pid := process.PIDs.Tgid
	db.processes[pid] = process
	entryLeaderPID := db.evaluateEntryLeader(process)
	if entryLeaderPID != nil {
		db.entryLeaderRelationships[pid] = *entryLeaderPID
		db.logger.Debugf("%v name: %s, entry_leader: %d, entry_type: %s", process.PIDs, process.Filename, *entryLeaderPID, string(db.entryLeaders[*entryLeaderPID]))
	} else {
		db.logger.Debugf("%v name: %s, NO ENTRY LEADER", process.PIDs, process.Filename)
	}
}

// InsertExec adds an exec event
func (db *DB) InsertExec(exec types.ProcessExecEvent) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	proc := Process{
		PIDs:             pidInfoFromProto(exec.PIDs),
		Creds:            credInfoFromProto(exec.Creds),
		CTTY:             ttyDevFromProto(exec.CTTY),
		Argv:             exec.Argv,
		Cwd:              exec.CWD,
		Filename:         exec.Filename,
		procfsLookupFail: exec.ProcfsLookupFail,
		insertTime:       time.Now(),
	}
	if proc.procfsLookupFail {
		db.stats.procfsLookupFail.Add(1)
	}

	// check to see if an orphaned exit event maps to this exec event.
	// the out-of-order problem where we get the exit before the exec usually happens under load.
	// if we don't track orphaned processes like this, we'll never scrub them from the DB.
	if evt, ok := db.removalMap[proc.PIDs.Tgid]; ok {
		proc.ExitCode = evt.exitCode
		db.stats.resolvedOrphanExits.Add(1)
		db.logger.Debugf("resolved orphan exit for pid %d", proc.PIDs.Tgid)
		evt.startTime = proc.PIDs.StartTimeNS
		db.removalMap[proc.PIDs.Tgid] = evt
	}

	db.processes[exec.PIDs.Tgid] = proc
	entryLeaderPID := db.evaluateEntryLeader(proc)
	if entryLeaderPID != nil {
		db.entryLeaderRelationships[exec.PIDs.Tgid] = *entryLeaderPID
	}
}

func (db *DB) createEntryLeader(pid uint32, entryType EntryType) {
	db.entryLeaders[pid] = entryType
	db.logger.Debugf("created entry leader %d: %s, name: %s", pid, string(entryType), db.processes[pid].Filename)
}

// pid returned is a pointer type because it is possible no matching PID is found.
func (db *DB) evaluateEntryLeader(p Process) *uint32 {
	pid := p.PIDs.Tgid

	// init never has an entry leader or meta type
	if p.PIDs.Tgid == 1 {
		db.logger.Debugf("entry_eval %d: process is init, no entry type", p.PIDs.Tgid)
		return nil
	}

	// kernel threads also never have an entry leader or meta type kthreadd
	// (always pid 2) is the parent of all kernel threads, by filtering pid ==
	// 2 || ppid == 2, we get rid of all of them
	if p.PIDs.Tgid == 2 || p.PIDs.Ppid == 2 {
		db.logger.Debugf("entry_eval %d: kernel threads never an entry type (parent is pid 2)", p.PIDs.Tgid)
		return nil
	}

	// could be an entry leader
	if p.PIDs.Tgid == p.PIDs.Sid {
		ttyType := tty.GetTTYType(p.CTTY.Major, p.CTTY.Minor)

		procBasename := basename(p.Filename)
		switch {
		case ttyType == tty.TTY:
			db.createEntryLeader(pid, Terminal)
			db.logger.Debugf("entry_eval %d: entry type is terminal", p.PIDs.Tgid)
			return &pid
		case ttyType == tty.TTYConsole && procBasename == "login":
			db.createEntryLeader(pid, EntryConsole)
			db.logger.Debugf("entry_eval %d: entry type is console", p.PIDs.Tgid)
			return &pid
		case p.PIDs.Ppid == 1:
			db.createEntryLeader(pid, Init)
			db.logger.Debugf("entry_eval %d: entry type is init", p.PIDs.Tgid)
			return &pid
		case !isFilteredExecutable(procBasename):
			if parent, ok := db.processes[p.PIDs.Ppid]; ok {
				parentBasename := basename(parent.Filename)
				if ttyType == tty.Pts && parentBasename == "ssm-session-worker" {
					db.createEntryLeader(pid, Ssm)
					db.logger.Debugf("entry_eval %d: entry type is ssm", p.PIDs.Tgid)
					return &pid
				} else if parentBasename == "sshd" && procBasename != "sshd" {
					// TODO: get ip from env vars
					db.createEntryLeader(pid, Sshd)
					db.logger.Debugf("entry_eval %d: entry type is sshd", p.PIDs.Tgid)
					return &pid
				} else if isContainerRuntime(parentBasename) {
					db.createEntryLeader(pid, Container)
					db.logger.Debugf("entry_eval %d: entry type is container", p.PIDs.Tgid)
					return &pid
				}
			}
		default:
			db.logger.Debugf("entry_eval %d: is a filtered executable: %s", p.PIDs.Tgid, procBasename)
		}
	}

	// if not a session leader or was not determined to be an entry leader, get
	// it via parent, session leader, group leader (in that order)
	relations := []struct {
		pid  uint32
		name string
	}{
		{
			pid:  p.PIDs.Ppid,
			name: "parent",
		},
		{
			pid:  p.PIDs.Sid,
			name: "session_leader",
		},
		{
			pid:  p.PIDs.Pgid,
			name: "group_leader",
		},
	}

	for _, relation := range relations {
		if entry, ok := db.entryLeaderRelationships[relation.pid]; ok {
			entryType := db.entryLeaders[entry]
			db.logger.Debugf("entry_eval %d: got entry_leader: %d (%s), from relative: %d (%s)", p.PIDs.Tgid, entry, string(entryType), relation.pid, relation.name)
			return &entry
		} else {
			db.logger.Debugf("entry_eval %d: failed to find relative: %d (%s)", p.PIDs.Tgid, relation.pid, relation.name)
		}
	}

	// if it's a session leader, then make it its own entry leader with unknown
	// entry type
	if p.PIDs.Tgid == p.PIDs.Sid {
		db.createEntryLeader(pid, EntryUnknown)
		db.logger.Debugf("entry_eval %d: this is a session leader and no relative has an entry leader. entry type is unknown", p.PIDs.Tgid)
		return &pid
	}

	db.logger.Debugf("entry_eval %d: this is not a session leader and no relative has an entry leader, entry_leader will be unset", p.PIDs.Tgid)
	return nil
}

// InsertSetsid adds a set SID event
func (db *DB) InsertSetsid(setsid types.ProcessSetsidEvent) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if entry, ok := db.processes[setsid.PIDs.Tgid]; ok {
		entry.PIDs = pidInfoFromProto(setsid.PIDs)
		db.processes[setsid.PIDs.Tgid] = entry
	} else {
		db.processes[setsid.PIDs.Tgid] = Process{
			PIDs: pidInfoFromProto(setsid.PIDs),
		}
	}
}

// InsertExit adds a process exit event
func (db *DB) InsertExit(exit types.ProcessExitEvent) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	pid := exit.PIDs.Tgid
	newRemoval := removalCandidate{
		pid:      pid,
		exitTime: time.Now(),
		exitCode: exit.ExitCode,
	}

	process, ok := db.processes[pid]
	if !ok {
		newRemoval.orphanTime = time.Now()
		db.logger.Debugf("pid %v for exit event not found in db, adding as orphan", pid)
	} else {
		// If we already have the process, add our exit info
		process.ExitCode = exit.ExitCode
		db.processes[pid] = process
		newRemoval.startTime = process.PIDs.StartTimeNS
	}
	db.removalMap[pid] = newRemoval
}

func fullProcessFromDBProcess(p Process) types.Process {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(p.PIDs.StartTimeNS)
	interactive := tty.InteractiveFromTTY(p.CTTY)

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
	ret.TTY.CharDevice.Major = uint16(p.CTTY.Major)
	ret.TTY.CharDevice.Minor = uint16(p.CTTY.Minor)
	ret.ExitCode = p.ExitCode

	return ret
}

func fillParent(process *types.Process, parent Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(parent.PIDs.StartTimeNS)

	interactive := tty.InteractiveFromTTY(parent.CTTY)
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

	interactive := tty.InteractiveFromTTY(groupLeader.CTTY)
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

	interactive := tty.InteractiveFromTTY(sessionLeader.CTTY)
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

	interactive := tty.InteractiveFromTTY(entryLeader.CTTY)
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

func (db *DB) setEntityID(process *types.Process) {
	if process.PID != 0 && process.Start != nil {
		process.EntityID = db.calculateEntityIDv1(process.PID, *process.Start)
	}

	if process.Parent.PID != 0 && process.Parent.Start != nil {
		process.Parent.EntityID = db.calculateEntityIDv1(process.Parent.PID, *process.Parent.Start)
	}

	if process.GroupLeader.PID != 0 && process.GroupLeader.Start != nil {
		process.GroupLeader.EntityID = db.calculateEntityIDv1(process.GroupLeader.PID, *process.GroupLeader.Start)
	}

	if process.SessionLeader.PID != 0 && process.SessionLeader.Start != nil {
		process.SessionLeader.EntityID = db.calculateEntityIDv1(process.SessionLeader.PID, *process.SessionLeader.Start)
	}

	if process.EntryLeader.PID != 0 && process.EntryLeader.Start != nil {
		process.EntryLeader.EntityID = db.calculateEntityIDv1(process.EntryLeader.PID, *process.EntryLeader.Start)
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

func (db *DB) HasProcess(pid uint32) bool {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	_, ok := db.processes[pid]
	return ok
}

func (db *DB) GetProcess(pid uint32) (types.Process, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	process, ok := db.processes[pid]
	if !ok {
		db.stats.failedToFindProcessCount.Add(1)
		return types.Process{}, errors.New("process not found")
	}
	db.stats.servedProcessCount.Add(1)

	ret := fullProcessFromDBProcess(process)

	if process.PIDs.Ppid != 0 {
		for i := 0; i < retryCount; i++ {
			if parent, ok := db.processes[process.PIDs.Ppid]; ok {
				fillParent(&ret, parent)
				break
			}
		}
	}

	if process.PIDs.Pgid != 0 {
		for i := 0; i < retryCount; i++ {
			if groupLeader, ok := db.processes[process.PIDs.Pgid]; ok {
				fillGroupLeader(&ret, groupLeader)
				break
			}
		}
	}

	if process.PIDs.Sid != 0 {
		for i := 0; i < retryCount; i++ {
			if sessionLeader, ok := db.processes[process.PIDs.Sid]; ok {
				fillSessionLeader(&ret, sessionLeader)
				break
			}
		}
	}

	if entryLeaderPID, foundEntryLeaderPID := db.entryLeaderRelationships[process.PIDs.Tgid]; foundEntryLeaderPID {
		if entryLeader, foundEntryLeader := db.processes[entryLeaderPID]; foundEntryLeader {
			// if there is an entry leader then there is a matching member in the entryLeaders table
			fillEntryLeader(&ret, db.entryLeaders[entryLeaderPID], entryLeader)
		} else {
			db.logger.Debugf("failed to find entry leader entry %d for %d (%s)", entryLeaderPID, pid, db.processes[pid].Filename)
		}
	} else {
		db.logger.Debugf("failed to find entry leader for %d (%s)", pid, db.processes[pid].Filename)
		db.stats.entryLeaderLookupFail.Add(1)
	}

	db.setEntityID(&ret)
	setSameAsProcess(&ret)

	return ret, nil
}

func (db *DB) GetEntryType(pid uint32) (EntryType, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if entryType, ok := db.entryLeaders[pid]; ok {
		return entryType, nil
	}
	return EntryUnknown, nil
}

func (db *DB) ScrapeProcfs() []uint32 {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	procs, err := db.procfs.GetAllProcesses()
	if err != nil {
		db.logger.Errorf("failed to get processes from procfs: %v", err)
		return make([]uint32, 0)
	}

	// sorting the slice to make sure that parents, session leaders, group
	// leaders come first in the queue
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].PIDs.Tgid == procs[j].PIDs.Ppid ||
			procs[i].PIDs.Tgid == procs[j].PIDs.Sid ||
			procs[i].PIDs.Tgid == procs[j].PIDs.Pgid
	})

	pids := make([]uint32, 0)
	for _, procInfo := range procs {
		process := Process{
			PIDs:     pidInfoFromProto(procInfo.PIDs),
			Creds:    credInfoFromProto(procInfo.Creds),
			CTTY:     ttyDevFromProto(procInfo.CTTY),
			Argv:     procInfo.Argv,
			Cwd:      procInfo.Cwd,
			Filename: procInfo.Filename,
		}

		db.insertProcess(process)
		pids = append(pids, process.PIDs.Tgid)
	}

	return pids
}

func stringStartsWithEntryInList(str string, list []string) bool {
	for _, entry := range list {
		if strings.HasPrefix(str, entry) {
			return true
		}
	}

	return false
}

func isContainerRuntime(executable string) bool {
	return slices.ContainsFunc(containerRuntimes[:], func(s string) bool {
		return strings.HasPrefix(executable, s)
	})
}

func isFilteredExecutable(executable string) bool {
	return stringStartsWithEntryInList(executable, filteredExecutables[:])
}

func (db *DB) Close() {
	close(db.stopChan)
}
