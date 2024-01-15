// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/bits"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/pkg/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/pkg/timeutils"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/types"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Process struct {
	Pids             types.PidInfo
	Creds            types.CredInfo
	CTty             types.TtyDev
	Argv             []string
	Cwd              string
	Env              map[string]string
	Filename         string
}

var (
	// The contents of these two files are needed to calculate entity IDs.
	// Fail fast on startup if we can't read them.
	bootID     = mustReadBootID()
	pidNsInode = mustReadPidNsInode()
	capNames   = []string{
		"CAP_CHOWN",              // 0
		"CAP_DAC_OVERRIDE",       // 1
		"CAP_DAC_READ_SEARCH",    // 2
		"CAP_FOWNER",             // 3
		"CAP_FSETID",             // 4
		"CAP_KILL",               // 5
		"CAP_SETGID",             // 6
		"CAP_SETUID",             // 7
		"CAP_SETPCAP",            // 8
		"CAP_LINUX_IMMUTABLE",    // 9
		"CAP_NET_BIND_SERVICE",   // 10
		"CAP_NET_BROADCAST",      // 11
		"CAP_NET_ADMIN",          // 12
		"CAP_NET_RAW",            // 13
		"CAP_IPC_LOCK",           // 14
		"CAP_IPC_OWNER",          // 15
		"CAP_SYS_MODULE",         // 16
		"CAP_SYS_RAWIO",          // 17
		"CAP_SYS_CHROOT",         // 18
		"CAP_SYS_PTRACE",         // 19
		"CAP_SYS_PACCT",          // 20
		"CAP_SYS_ADMIN",          // 21
		"CAP_SYS_BOOT",           // 22
		"CAP_SYS_NICE",           // 23
		"CAP_SYS_RESOURCE",       // 24
		"CAP_SYS_TIME",           // 25
		"CAP_SYS_TTY_CONFIG",     // 26
		"CAP_MKNOD",              // 27
		"CAP_LEASE",              // 28
		"CAP_AUDIT_WRITE",        // 29
		"CAP_AUDIT_CONTROL",      // 30
		"CAP_SETFCAP",            // 31
		"CAP_MAC_OVERRIDE",       // 32
		"CAP_MAC_ADMIN",          // 33
		"CAP_SYSLOG",             // 34
		"CAP_WAKE_ALARM",         // 35
		"CAP_BLOCK_SUSPEND",      // 36
		"CAP_AUDIT_READ",         // 37
		"CAP_PERFMON",            // 38
		"CAP_BPF",                // 39
		"CAP_CHECKPOINT_RESTORE", // 40
		// The ECS spec allows for numerical string representation.
		// The following capability values are not assigned as of Dec 28, 2023.
		// If they are added in a future kernel, and this slice has not been
		// updated, the numerical string will used.
		"41",
		"42",
		"43",
		"44",
		"45",
		"46",
		"47",
		"48",
		"49",
		"50",
		"51",
		"52",
		"53",
		"54",
		"55",
		"56",
		"57",
		"58",
		"59",
		"60",
		"61",
		"62",
		"63",
	}
)

func mustReadBootID() string {
	bootID, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		panic(fmt.Sprintf("could not read /proc/sys/kernel/random/boot_id: %v", err))
	}

	return strings.TrimRight(string(bootID), "\n")
}

func mustReadPidNsInode() uint64 {
	var ret uint64

	pidNsInodeRaw, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		panic(fmt.Sprintf("could not read /proc/self/ns/pid: %v", err))
	}

	if _, err = fmt.Sscanf(pidNsInodeRaw, "pid:[%d]", &ret); err != nil {
		panic(fmt.Sprintf("could not parse contents of /proc/self/ns/pid (%s): %v", pidNsInodeRaw, err))
	}

	return ret
}

func pidInfoFromProto(p types.PidInfo) types.PidInfo {
	return types.PidInfo{
		StartTimeNs: p.StartTimeNs,
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

func ttyTermiosFromProto(p types.TtyTermios) types.TtyTermios {
	return types.TtyTermios{
		CIflag: p.CIflag,
		COflag: p.COflag,
		CLflag: p.CLflag,
		CCflag: p.CCflag,
	}
}

func ttyWinsizeFromProto(p types.TtyWinsize) types.TtyWinsize {
	return types.TtyWinsize{
		Rows: p.Rows,
		Cols: p.Cols,
	}
}

func ttyDevFromProto(p types.TtyDev) types.TtyDev {
	return types.TtyDev{
		Major:   p.Major,
		Minor:   p.Minor,
		Winsize: ttyWinsizeFromProto(p.Winsize),
		Termios: ttyTermiosFromProto(p.Termios),
	}
}

type SimpleDB struct {
	sync.RWMutex
	logger                   *logp.Logger
	processes                map[uint32]Process
	entryLeaders             map[uint32]EntryType
	entryLeaderRelationships map[uint32]uint32
	procfs                   procfs.Reader
}

func NewSimpleDB(reader procfs.Reader, logger logp.Logger) *SimpleDB {
	ret := &SimpleDB{
		logger:                   logp.NewLogger("processdb"),
		processes:                make(map[uint32]Process),
		entryLeaders:             make(map[uint32]EntryType),
		entryLeaderRelationships: make(map[uint32]uint32),
		procfs:                   reader,
	}

	return ret
}

func (db *SimpleDB) calculateEntityIDv1(pid uint32, startTime time.Time) string {
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

func (db *SimpleDB) InsertFork(fork types.ProcessForkEvent) error {
	db.Lock()
	defer db.Unlock()

	pid := fork.ChildPids.Tgid
	ppid := fork.ParentPids.Tgid
	if entry, ok := db.processes[ppid]; ok {
		entry.Pids = pidInfoFromProto(fork.ChildPids)
		entry.Creds = credInfoFromProto(fork.Creds)
		db.processes[pid] = entry
		if entryPid, ok := db.entryLeaderRelationships[ppid]; ok {
			db.entryLeaderRelationships[pid] = entryPid
		}
	} else {
		db.processes[pid] = Process{
			Pids:  pidInfoFromProto(fork.ChildPids),
			Creds: credInfoFromProto(fork.Creds),
		}
	}

	return nil
}

func (db *SimpleDB) insertProcess(process Process) {
	pid := process.Pids.Tgid
	db.processes[pid] = process
	entryLeaderPid := db.evaluateEntryLeader(process)
	if entryLeaderPid != nil {
		db.entryLeaderRelationships[pid] = *entryLeaderPid
		db.logger.Debugf("%v name: %s, entry_leader: %d, entry_type: %s", process.Pids, process.Filename, *entryLeaderPid, string(db.entryLeaders[*entryLeaderPid]))
	} else {
		db.logger.Debugf("%v name: %s, NO ENTRY LEADER", process.Pids, process.Filename)
	}
}

func (db *SimpleDB) InsertExec(exec types.ProcessExecEvent) error {
	db.Lock()
	defer db.Unlock()

	proc := Process{
		Pids:             pidInfoFromProto(exec.Pids),
		Creds:            credInfoFromProto(exec.Creds),
		CTty:             ttyDevFromProto(exec.CTty),
		Argv:             exec.Argv,
		Cwd:              exec.Cwd,
		Env:              exec.Env,
		Filename:         exec.Filename,
	}

	db.processes[exec.Pids.Tgid] = proc
	entryLeaderPid := db.evaluateEntryLeader(proc)
	if entryLeaderPid != nil {
		db.entryLeaderRelationships[exec.Pids.Tgid] = *entryLeaderPid
	}

	return nil
}

func (db *SimpleDB) createEntryLeader(pid uint32, entryType EntryType) {
	db.entryLeaders[pid] = entryType
	db.logger.Debugf("created entry leader %d: %s, name: %s", pid, string(entryType), db.processes[pid].Filename)
}

// pid returned is a pointer type because its possible for no
func (db *SimpleDB) evaluateEntryLeader(p Process) *uint32 {
	pid := p.Pids.Tgid

	// init never has an entry leader or meta type
	if p.Pids.Tgid == 1 {
		db.logger.Debugf("entry_eval %d: process is init, no entry type", p.Pids.Tgid)
		return nil
	}

	// kernel threads also never have an entry leader or meta type kthreadd
	// (always pid 2) is the parent of all kernel threads, by filtering pid ==
	// 2 || ppid == 2, we get rid of all of them
	if p.Pids.Tgid == 2 || p.Pids.Ppid == 2 {
		db.logger.Debugf("entry_eval %d: kernel threads never an entry type (parent is pid 2)", p.Pids.Tgid)
		return nil
	}

	// could be an entry leader
	if p.Pids.Tgid == p.Pids.Sid {
		ttyType := getTtyType(p.CTty.Major, p.CTty.Minor)

		procBasename := basename(p.Filename)
		if ttyType == Tty {
			db.createEntryLeader(pid, Terminal)
			db.logger.Debugf("entry_eval %d: entry type is terminal", p.Pids.Tgid)
			return &pid
		} else if ttyType == TtyConsole && procBasename == "login" {
			db.createEntryLeader(pid, EntryConsole)
			db.logger.Debugf("entry_eval %d: entry type is console", p.Pids.Tgid)
			return &pid
		} else if p.Pids.Ppid == 1 {
			db.createEntryLeader(pid, Init)
			db.logger.Debugf("entry_eval %d: entry type is init", p.Pids.Tgid)
			return &pid
		} else if !isFilteredExecutable(procBasename) {
			if parent, ok := db.processes[p.Pids.Ppid]; ok {
				parentBasename := basename(parent.Filename)
				if ttyType == Pts && parentBasename == "ssm-session-worker" {
					db.createEntryLeader(pid, Ssm)
					db.logger.Debugf("entry_eval %d: entry type is ssm", p.Pids.Tgid)
					return &pid
				} else if parentBasename == "sshd" && procBasename != "sshd" {
					// TODO: get ip from env vars
					db.createEntryLeader(pid, Sshd)
					db.logger.Debugf("entry_eval %d: entry type is sshd", p.Pids.Tgid)
					return &pid
				} else if isContainerRuntime(parentBasename) {
					db.createEntryLeader(pid, Container)
					db.logger.Debugf("entry_eval %d: entry type is container", p.Pids.Tgid)
					return &pid
				}
			}
		} else {
			db.logger.Debugf("entry_eval %d: is a filtered executable: %s", p.Pids.Tgid, procBasename)
		}
	}

	// if not a session leader or was not determined to be an entry leader, get
	// it via parent, session leader, group leader (in that order)
	relations := []struct {
		pid  uint32
		name string
	}{
		{
			pid:  p.Pids.Ppid,
			name: "parent",
		},
		{
			pid:  p.Pids.Sid,
			name: "session_leader",
		},
		{
			pid:  p.Pids.Pgid,
			name: "group_leader",
		},
	}

	for _, relation := range relations {
		if entry, ok := db.entryLeaderRelationships[relation.pid]; ok {
			entryType := db.entryLeaders[entry]
			db.logger.Debugf("entry_eval %d: got entry_leader: %d (%s), from relative: %d (%s)", p.Pids.Tgid, entry, string(entryType), relation.pid, relation.name)
			return &entry
		} else {
			db.logger.Debugf("entry_eval %d: failed to find relative: %d (%s)", p.Pids.Tgid, relation.pid, relation.name)
		}
	}

	// if it's a session leader, then make it its own entry leader with unknown
	// entry type
	if p.Pids.Tgid == p.Pids.Sid {
		db.createEntryLeader(pid, EntryUnknown)
		db.logger.Debugf("entry_eval %d: this is a session leader and no relative has an entry leader. entry type is unknown", p.Pids.Tgid)
		return &pid
	}

	db.logger.Debugf("entry_eval %d: this is not a session leader and no relative has an entry leader, entry_leader will be unset", p.Pids.Tgid)
	return nil
}

func (db *SimpleDB) InsertSetsid(setsid types.ProcessSetsidEvent) error {
	db.Lock()
	defer db.Unlock()

	if entry, ok := db.processes[setsid.Pids.Tgid]; ok {
		entry.Pids = pidInfoFromProto(setsid.Pids)
		db.processes[setsid.Pids.Tgid] = entry
	} else {
		db.processes[setsid.Pids.Tgid] = Process{
			Pids: pidInfoFromProto(setsid.Pids),
		}
	}

	return nil
}

func (db *SimpleDB) InsertExit(exit types.ProcessExitEvent) error {
	db.Lock()
	defer db.Unlock()

	pid := exit.Pids.Tgid
	delete(db.processes, pid)
	delete(db.entryLeaders, pid)
	delete(db.entryLeaderRelationships, pid)
	return nil
}

// TODO: is this the correct definition? I looked in endpoint and I swear it looks too simple/generalized
func interactiveFromTty(tty types.TtyDev) bool {
	return TtyUnknown != getTtyType(tty.Major, tty.Minor)
}

func ecsCapsFromU64(capabilities uint64) []string {
	var ecsCaps []string
	if c := bits.OnesCount64(capabilities); c > 0 {
		ecsCaps = make([]string, 0, c)
	}
	for bitnum := 0; bitnum < 64; bitnum++ {
		if (capabilities & (1 << bitnum)) > 0 {
			ecsCaps = append(ecsCaps, capNames[bitnum])
		}
	}
	return ecsCaps
}

func fullProcessFromDBProcess(p Process) types.Process {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(p.Pids.StartTimeNs)
	interactive := interactiveFromTty(p.CTty)

	ret := types.Process{
		PID:              p.Pids.Tgid,
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
	ret.Group.ID = strconv.FormatUint(uint64(egid), 10)
	ret.Thread.Capabilities.Permitted = ecsCapsFromU64(p.Creds.CapPermitted)
	ret.Thread.Capabilities.Effective = ecsCapsFromU64(p.Creds.CapEffective)

	return ret
}

func fillParent(process *types.Process, parent Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(parent.Pids.StartTimeNs)

	interactive := interactiveFromTty(parent.CTty)
	euid := parent.Creds.Euid
	egid := parent.Creds.Egid
	process.Parent.PID = parent.Pids.Tgid
	process.Parent.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.Parent.Name = basename(parent.Filename)
	process.Parent.Executable = parent.Filename
	process.Parent.Args = parent.Argv
	process.Parent.WorkingDirectory = parent.Cwd
	process.Parent.Interactive = &interactive
	process.Parent.User.ID = strconv.FormatUint(uint64(euid), 10)
	process.Parent.Group.ID = strconv.FormatUint(uint64(egid), 10)
}

func fillGroupLeader(process *types.Process, groupLeader Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(groupLeader.Pids.StartTimeNs)

	interactive := interactiveFromTty(groupLeader.CTty)
	euid := groupLeader.Creds.Euid
	egid := groupLeader.Creds.Egid
	process.GroupLeader.PID = groupLeader.Pids.Tgid
	process.GroupLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.GroupLeader.Name = basename(groupLeader.Filename)
	process.GroupLeader.Executable = groupLeader.Filename
	process.GroupLeader.Args = groupLeader.Argv
	process.GroupLeader.WorkingDirectory = groupLeader.Cwd
	process.GroupLeader.Interactive = &interactive
	process.GroupLeader.User.ID = strconv.FormatUint(uint64(euid), 10)
	process.GroupLeader.Group.ID = strconv.FormatUint(uint64(egid), 10)
}

func fillSessionLeader(process *types.Process, sessionLeader Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(sessionLeader.Pids.StartTimeNs)

	interactive := interactiveFromTty(sessionLeader.CTty)
	euid := sessionLeader.Creds.Euid
	egid := sessionLeader.Creds.Egid
	process.SessionLeader.PID = sessionLeader.Pids.Tgid
	process.SessionLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.SessionLeader.Name = basename(sessionLeader.Filename)
	process.SessionLeader.Executable = sessionLeader.Filename
	process.SessionLeader.Args = sessionLeader.Argv
	process.SessionLeader.WorkingDirectory = sessionLeader.Cwd
	process.SessionLeader.Interactive = &interactive
	process.SessionLeader.User.ID = strconv.FormatUint(uint64(euid), 10)
	process.SessionLeader.Group.ID = strconv.FormatUint(uint64(egid), 10)
}

func fillEntryLeader(process *types.Process, entryType EntryType, entryLeader Process) {
	reducedPrecisionStartTime := timeutils.ReduceTimestampPrecision(entryLeader.Pids.StartTimeNs)

	interactive := interactiveFromTty(entryLeader.CTty)
	euid := entryLeader.Creds.Euid
	egid := entryLeader.Creds.Egid
	process.EntryLeader.PID = entryLeader.Pids.Tgid
	process.EntryLeader.Start = timeutils.TimeFromNsSinceBoot(reducedPrecisionStartTime)
	process.EntryLeader.Name = basename(entryLeader.Filename)
	process.EntryLeader.Executable = entryLeader.Filename
	process.EntryLeader.Args = entryLeader.Argv
	process.EntryLeader.WorkingDirectory = entryLeader.Cwd
	process.EntryLeader.Interactive = &interactive
	process.EntryLeader.User.ID = strconv.FormatUint(uint64(euid), 10)
	process.EntryLeader.Group.ID = strconv.FormatUint(uint64(egid), 10)

	process.EntryLeader.EntryMeta.Type = string(entryType)
}

func (db *SimpleDB) setEntityID(process *types.Process) {
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

func (db *SimpleDB) GetProcess(pid uint32) (types.Process, error) {
	db.RLock()
	defer db.RUnlock()

	process, ok := db.processes[pid]
	if !ok {
		return types.Process{}, errors.New("process not found")
	}

	ret := fullProcessFromDBProcess(process)

	if parent, ok := db.processes[process.Pids.Ppid]; ok {
		fillParent(&ret, parent)
	}

	if groupLeader, ok := db.processes[process.Pids.Pgid]; ok {
		fillGroupLeader(&ret, groupLeader)
	}

	if sessionLeader, ok := db.processes[process.Pids.Sid]; ok {
		fillSessionLeader(&ret, sessionLeader)
	}

	if entryLeaderPid, foundEntryLeaderPid := db.entryLeaderRelationships[process.Pids.Tgid]; foundEntryLeaderPid {
		if entryLeader, foundEntryLeader := db.processes[entryLeaderPid]; foundEntryLeader {
			// if there is an entry leader then there is a matching member in the entryLeaders table
			fillEntryLeader(&ret, db.entryLeaders[entryLeaderPid], entryLeader)
		} else {
			db.logger.Errorf("failed to find entry leader entry %d for %d (%s)", entryLeaderPid, pid, db.processes[pid].Filename)
		}
	} else {
		db.logger.Errorf("failed to find entry leader for %d (%s)", pid, db.processes[pid].Filename)
	}

	db.setEntityID(&ret)
	setSameAsProcess(&ret)

	return ret, nil
}

func (db *SimpleDB) GetEntryType(pid uint32) (EntryType, error) {
	db.RLock()
	defer db.RUnlock()

	if entryType, ok := db.entryLeaders[pid]; ok {
		return entryType, nil
	}
	return EntryUnknown, nil
}

func (db *SimpleDB) ScrapeProcfs() []uint32 {
	db.Lock()
	defer db.Unlock()

	procs, err := db.procfs.GetAllProcesses()
	if err != nil {
		db.logger.Errorf("failed to get processes from procfs: %v", err)
		return make([]uint32, 0)
	}

	// sorting the slice to make sure that parents, session leaders, group
	// leaders come first in the queue
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].Pids.Tgid == procs[j].Pids.Ppid ||
			procs[i].Pids.Tgid == procs[j].Pids.Sid ||
			procs[i].Pids.Tgid == procs[j].Pids.Pgid
	})

	pids := make([]uint32, 0)
	for _, procInfo := range procs {
		process := Process{
			Pids:             pidInfoFromProto(procInfo.Pids),
			Creds:            credInfoFromProto(procInfo.Creds),
			CTty:             ttyDevFromProto(procInfo.CTty),
			Argv:             procInfo.Argv,
			Cwd:              procInfo.Cwd,
			Env:              procInfo.Env,
			Filename:         procInfo.Filename,
		}

		db.insertProcess(process)
		pids = append(pids, process.Pids.Tgid)
	}

	return pids
}
