// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
)

const (
	containerdShimPath = "/bin/containerd-shim-runc-v2"
	containerdPath     = "/bin/containerd"
	sshdPath           = "/usr/bin/sshd"
	lsPath             = "/usr/bin/ls"
	bashPath           = "/usr/bin/bash"
	grepPath           = "/usr/bin/grep"
)

// Entry evaluation tests
//
// The entry leader isn't an entirely rigorous conceptual framework but that
// shortcoming is outweighted by the large and immediate value it provides.
//
// The idea is to assign two pieces of data to each process, the "entry meta"
// and "entry leader", the former of which describes how the user or system
// that was ultimately responsible for executing this process got into to the
// box (e.g. ssh, ssm, kubectl exec) and the latter of which describes the
// process associated with the user or system's initial entry into the "box"
// (be it a container, VM or otherwise).
//
// Generally speaking, the first session leader in a process lineage of an
// interactive session is an entry leader having an entry meta type depending
// on its lineage. For example, in the following process tree, "bash" is an
// entry leader with entry meta type "sshd":
//
// systemd            (pid 1 sid 1)
// \___ sshd          (pid 100 sid 100)
//      \___ bash     (pid 1000 sid 1000)
//           \___ vim (pid 1001 sid 1000)
//
// Further entry meta types exist for ssm, container runtimes, serial consoles
// and other ways to get into a "box" (be it a container or actual machine).
// The entry meta type "init" is assigned to system processes created by the
// init service (e.g. rsyslogd, sshd).
//
// As should probably be apparent, the code to assign an entry meta type to a
// process is essentially a large amount of conditional logic with a ton of
// edge cases. It's something we "bolt on" to the linux process model, and thus
// finicky and highly subject to bugs.
//
// Thankfully, writing unit tests for entry leader evaluation is rather
// straightforward as it's basically a pure function that requires no external
// infrastructure to test (just create a mock process event with your desired
// fields set and pass it in).
//
// These tests should effectively serve as the spec for how we assign entry
// leaders. When further entry meta types or cases are added, tests should be

func requireProcess(t *testing.T, db *DB, pid uint32, processPath string) {
	t.Helper()
	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	require.Equal(t, pid, process.PID)
	require.Equal(t, processPath, process.Executable)
	if processPath == "" {
		require.Equal(t, "", process.Name)
	} else {
		require.Equal(t, path.Base(processPath), process.Name)
	}
}

func requireParent(t *testing.T, db *DB, pid uint32, ppid uint32) {
	t.Helper()
	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	require.Equal(t, ppid, process.Parent.PID)
}

func requireParentUnset(t *testing.T, process types.Process) {
	t.Helper()
	require.Equal(t, "", process.Parent.EntityID)
	require.Equal(t, uint32(0), process.Parent.PID)
	require.Nil(t, process.Parent.Start)
}

func requireSessionLeader(t *testing.T, db *DB, pid uint32, sid uint32) {
	t.Helper()
	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	require.Equal(t, sid, process.SessionLeader.PID)
	require.NotNil(t, process.SessionLeader.SameAsProcess)
	require.Equal(t, pid == sid, *process.SessionLeader.SameAsProcess)
}

func requireSessionLeaderUnset(t *testing.T, process types.Process) {
	t.Helper()
	require.Equal(t, "", process.SessionLeader.EntityID)
	require.Equal(t, uint32(0), process.SessionLeader.PID)
	require.Nil(t, process.SessionLeader.Start)
}

func requireGroupLeader(t *testing.T, db *DB, pid uint32, pgid uint32) {
	t.Helper()
	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	require.Equal(t, pgid, process.GroupLeader.PID)
	require.NotNil(t, process.GroupLeader.SameAsProcess)
	require.Equal(t, pid == pgid, *process.GroupLeader.SameAsProcess)
}

func requireEntryLeader(t *testing.T, db *DB, pid uint32, entryPid uint32, expectedEntryType EntryType) {
	t.Helper()
	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	require.Equal(t, entryPid, process.EntryLeader.PID)
	require.NotNil(t, process.EntryLeader.SameAsProcess)
	require.Equal(t, pid == entryPid, *process.EntryLeader.SameAsProcess)

	entryType, err := db.GetEntryType(entryPid)
	require.Nil(t, err)
	require.Equal(t, expectedEntryType, entryType)
}

func requireEntryLeaderUnset(t *testing.T, process types.Process) {
	t.Helper()
	require.Equal(t, "", process.EntryLeader.EntityID)
	require.Equal(t, uint32(0), process.EntryLeader.PID)
	require.Nil(t, process.EntryLeader.Start)
}

// tries to construct fork event from what's in the db
func insertForkAndExec(t *testing.T, db *DB, exec types.ProcessExecEvent) {
	t.Helper()
	var fork types.ProcessForkEvent
	fork.ChildPids = exec.Pids
	parent, err := db.GetProcess(exec.Pids.Ppid)
	if err != nil {
		fork.ParentPids = exec.Pids
		fork.ParentPids.Tgid = exec.Pids.Ppid
		fork.ParentPids.Ppid = 0
		fork.ParentPids.Pgid = 0

		fork.ChildPids.Pgid = exec.Pids.Ppid

		// if the exec makes itself a session and the parent is no where to be
		// found we'll make the parent its own session
		if exec.Pids.Tgid == exec.Pids.Sid {
			fork.ParentPids.Sid = exec.Pids.Ppid
		}
	} else {
		fork.ParentPids.Tgid = parent.PID
		fork.ParentPids.Ppid = parent.Parent.PID
		fork.ParentPids.Sid = parent.SessionLeader.PID

		// keep group leader the same for now
		fork.ParentPids.Pgid = exec.Pids.Pgid
	}

	if fork.ParentPids.Tgid != 0 {
		db.InsertFork(fork)
	}

	db.InsertExec(exec)
}

var systemdPath = "/sbin/systemd"

func populateProcfsWithInit(reader *procfs.MockReader) {
	reader.AddEntry(1, procfs.ProcessInfo{
		Pids: types.PidInfo{
			Tid:  1,
			Tgid: 1,
			Pgid: 0,
			Sid:  1,
		},
		Filename: systemdPath,
	})
}

func TestSingleProcessSessionLeaderEntryTypeTerminal(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	pid := uint32(1234)
	procPath := "/bin/noproc"
	db.InsertExec(types.ProcessExecEvent{
		Filename: procPath,
		Pids: types.PidInfo{
			Tgid: pid,
			Sid:  pid,
		},
		CTty: types.TtyDev{
			Major: 4,
			Minor: 64,
		},
	})

	requireProcess(t, db, 1234, procPath)
	requireEntryLeader(t, db, 1234, 1234, Terminal)
}

func TestSingleProcessSessionLeaderLoginProcess(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	pid := uint32(1234)
	loginPath := "/bin/login"
	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: loginPath,
		Pids: types.PidInfo{
			Tgid: pid,
			Sid:  pid,
		},
		CTty: types.TtyDev{
			Major: 4,
			Minor: 62,
		},
	})

	process, err := db.GetProcess(1234)
	require.Nil(t, err)
	requireParentUnset(t, process)

	requireProcess(t, db, pid, "/bin/login")
	requireSessionLeader(t, db, pid, pid)
	requireEntryLeader(t, db, pid, pid, EntryConsole)
}

func TestSingleProcessSessionLeaderChildOfInit(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	pid := uint32(100)
	rsyslogdPath := "/bin/rsyslogd"
	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: rsyslogdPath,
		Pids: types.PidInfo{
			Tgid: pid,
			Sid:  pid,
			Ppid: 1,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	process, err := db.GetProcess(1234)
	require.NotNil(t, err)
	requireParentUnset(t, process)

	requireProcess(t, db, pid, rsyslogdPath)
	requireSessionLeader(t, db, pid, pid)
	requireEntryLeader(t, db, pid, pid, Init)
}

func TestSingleProcessSessionLeaderChildOfSsmSessionWorker(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	ssmPid := uint32(999)
	bashPid := uint32(1000)
	ssmPath := "/usr/bin/ssm-session-worker"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: ssmPath,
		Pids: types.PidInfo{
			Tgid: ssmPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: ssmPid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, ssmPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Ssm)
}

func TestSingleProcessSessionLeaderChildOfSshd(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshdPid := uint32(999)
	bashPid := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshdPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: sshdPid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, sshdPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Sshd)
}

func TestSingleProcessSessionLeaderChildOfContainerdShim(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	containerdShimPid := uint32(999)
	bashPid := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		Pids: types.PidInfo{
			Tgid: containerdShimPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: containerdShimPid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, containerdShimPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Container)
}

func TestSingleProcessSessionLeaderChildOfRunc(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	runcPid := uint32(999)
	bashPid := uint32(1000)
	runcPath := "/bin/runc"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: runcPath,
		Pids: types.PidInfo{
			Tgid: runcPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: runcPid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, runcPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Container)
}

func TestSingleProcessEmptyProcess(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	// No information in proc at all, entry type should be "unknown"
	// and entry leader pid should be unset (since pid is not set)
	pid := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: pid,
			Sid:  pid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	requireParentUnset(t, process)

	requireProcess(t, db, pid, bashPath)
	requireSessionLeader(t, db, pid, pid)
	requireEntryLeader(t, db, pid, pid, EntryUnknown)
}

// Entry evaluation code should overwrite an old EntryLeaderPid and
// EntryLeaderEntryMetaType
func TestSingleProcessOverwriteOldEntryLeader(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	ssmPid := uint32(999)
	bashPid := uint32(1000)
	ssmPath := "/usr/bin/ssm-session-worker"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: ssmPath,
		Pids: types.PidInfo{
			Tgid: ssmPid,
			Sid:  ssmPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  ssmPid,
			Ppid: ssmPid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	// bash is not a session leader so it shouldn't be an entry leader. Its
	// entry leader should be ssm, which is an init entry leader
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, ssmPid)
	requireSessionLeader(t, db, bashPid, ssmPid)
	requireEntryLeader(t, db, bashPid, ssmPid, Init)

	// skiping setsid event and assuming the pids will be updated in this exec
	db.InsertExec(types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: ssmPid,
		},
		CTty: types.TtyDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, ssmPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Ssm)
}

// /	                 (pid, sid, entry meta, entry leader)
//
// systemd               (1, 1, none, none)
//
//	\___ sshd            (100, 100, "init", 100)
//	      \___ bash      (1000, 1000, "sshd", 1000)
//	            \___ ls  (1001, 1000, "sshd", 1000)
//
// This is unrealistic, sshd usually forks a bunch of sshd children before
// exec'ing bash (see subsequent tests) but is theoretically possible and
// thus something we should handle.
func TestInitSshdBashLs(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshdPid := uint32(100)
	bashPid := uint32(1000)
	lsPid := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshdPid,
			Sid:  sshdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: sshdPid,
			Pgid: bashPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		Pids: types.PidInfo{
			Tgid: lsPid,
			Sid:  bashPid,
			Ppid: bashPid,
			Pgid: lsPid,
		},
	})

	// systemd
	systemd, err := db.GetProcess(1)
	require.Nil(t, err)
	requireParentUnset(t, systemd)
	requireEntryLeaderUnset(t, systemd)

	requireProcess(t, db, 1, systemdPath)
	requireSessionLeader(t, db, 1, 1)

	// sshd
	requireProcess(t, db, sshdPid, sshdPath)
	requireParent(t, db, sshdPid, 1)
	requireSessionLeader(t, db, sshdPid, sshdPid)
	requireEntryLeader(t, db, sshdPid, sshdPid, Init)

	// bash
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, sshdPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Sshd)
	requireGroupLeader(t, db, bashPid, bashPid)

	// ls
	requireProcess(t, db, lsPid, lsPath)
	requireParent(t, db, lsPid, bashPid)
	requireSessionLeader(t, db, lsPid, bashPid)
	requireEntryLeader(t, db, lsPid, bashPid, Sshd)
	requireGroupLeader(t, db, lsPid, lsPid)
}

// /                           (pid, sid, entry meta, entry leader)
//
// systemd                     (1, 1, none, none)
//
//	\___ sshd                  (100, 100, "init", 100)
//	      \___ sshd            (101, 101, "init", 100)
//	            \___ bash      (1000, 1000, "sshd", 1000)
//	                  \___ ls  (1001, 1000, "sshd", 1000)
//
// sshd will usually fork a bunch of sshd children before invoking a shell
// usually 2 if it's a root shell, or 3 if it's a non-root shell. All
// "intermediate" sshd's should have entry meta "init" and an entry leader
// pid of the topmost sshd.
func TestInitSshdSshdBashLs(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshd0Pid := uint32(100)
	sshd1Pid := uint32(101)
	bashPid := uint32(1000)
	lsPid := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshd0Pid,
			Sid:  sshd0Pid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshd1Pid,
			Sid:  sshd1Pid,
			Ppid: sshd0Pid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: sshd1Pid,
			Pgid: bashPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		Pids: types.PidInfo{
			Tgid: lsPid,
			Sid:  bashPid,
			Ppid: bashPid,
			Pgid: lsPid,
		},
	})

	// systemd
	systemd, err := db.GetProcess(1)
	require.Nil(t, err)
	requireParentUnset(t, systemd)
	requireEntryLeaderUnset(t, systemd)

	requireProcess(t, db, 1, systemdPath)
	requireSessionLeader(t, db, 1, 1)

	// sshd0
	requireProcess(t, db, sshd0Pid, sshdPath)
	requireParent(t, db, sshd0Pid, 1)
	requireSessionLeader(t, db, sshd0Pid, sshd0Pid)
	requireEntryLeader(t, db, sshd0Pid, sshd0Pid, Init)

	// sshd1
	requireProcess(t, db, sshd1Pid, sshdPath)
	requireParent(t, db, sshd1Pid, sshd0Pid)
	requireSessionLeader(t, db, sshd1Pid, sshd1Pid)
	requireEntryLeader(t, db, sshd1Pid, sshd0Pid, Init)

	// bash
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, sshd1Pid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Sshd)

	// ls
	requireProcess(t, db, lsPid, lsPath)
	requireParent(t, db, lsPid, bashPid)
	requireSessionLeader(t, db, lsPid, bashPid)
	requireEntryLeader(t, db, lsPid, bashPid, Sshd)
}

// /	                             (pid, sid, entry meta, entry leader)
// systemd                           (1, 1, none, none)
//
//	\___ sshd                        (100, 100, "init", 100)
//	      \___ sshd                  (101, 101, "init", 100)
//	            \___ sshd            (102, 101, "init", 100)
//	                  \___ bash      (1000, 1000, "sshd", 1000)
//	                        \___ ls  (1001, 1000, "sshd", 1000)
func TestInitSshdSshdSshdBashLs(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshd0Pid := uint32(100)
	sshd1Pid := uint32(101)
	sshd2Pid := uint32(102)
	bashPid := uint32(1000)
	lsPid := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshd0Pid,
			Sid:  sshd0Pid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshd1Pid,
			Sid:  sshd1Pid,
			Ppid: sshd0Pid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshd2Pid,
			Sid:  sshd1Pid,
			Ppid: sshd1Pid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: sshd2Pid,
			Pgid: bashPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		Pids: types.PidInfo{
			Tgid: lsPid,
			Sid:  bashPid,
			Ppid: bashPid,
			Pgid: lsPid,
		},
	})

	// systemd
	systemd, err := db.GetProcess(1)
	require.Nil(t, err)
	requireParentUnset(t, systemd)
	requireEntryLeaderUnset(t, systemd)

	requireProcess(t, db, 1, systemdPath)
	requireSessionLeader(t, db, 1, 1)

	// sshd0
	requireProcess(t, db, sshd0Pid, sshdPath)
	requireParent(t, db, sshd0Pid, 1)
	requireSessionLeader(t, db, sshd0Pid, sshd0Pid)
	requireEntryLeader(t, db, sshd0Pid, sshd0Pid, Init)

	// sshd1
	requireProcess(t, db, sshd1Pid, sshdPath)
	requireParent(t, db, sshd1Pid, sshd0Pid)
	requireSessionLeader(t, db, sshd1Pid, sshd1Pid)
	requireEntryLeader(t, db, sshd1Pid, sshd0Pid, Init)

	// sshd2
	requireProcess(t, db, sshd2Pid, sshdPath)
	requireParent(t, db, sshd2Pid, sshd1Pid)
	requireSessionLeader(t, db, sshd2Pid, sshd1Pid)
	requireEntryLeader(t, db, sshd2Pid, sshd0Pid, Init)

	// bash
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, sshd2Pid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Sshd)

	// ls
	requireProcess(t, db, lsPid, lsPath)
	requireParent(t, db, lsPid, bashPid)
	requireSessionLeader(t, db, lsPid, bashPid)
	requireEntryLeader(t, db, lsPid, bashPid, Sshd)
}

// /                                   (pid, sid, entry meta, entry leader)
//
// systemd
//
//	\___ containerd                    (100, 100, "init", 100)
//	      \___ containerd-shim-runc-v2 (1000, 100, "init", 100)
//
// containerd-shim-runc-v2 will reparent itself to init just prior to
// executing the containerized process.
func TestInitContainerdContainerdShim(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	containerdPid := uint32(100)
	containerdShimPid := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdPath,
		Pids: types.PidInfo{
			Tgid: containerdPid,
			Sid:  containerdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		Pids: types.PidInfo{
			Tgid: containerdShimPid,
			Sid:  containerdPid,
			Ppid: containerdPid,
		},
	})

	// containerd
	requireProcess(t, db, containerdPid, containerdPath)
	requireParent(t, db, containerdPid, 1)
	requireSessionLeader(t, db, containerdPid, containerdPid)
	requireEntryLeader(t, db, containerdPid, containerdPid, Init)

	// containerd-shim-runc-v2
	requireProcess(t, db, containerdShimPid, containerdShimPath)
	requireParent(t, db, containerdShimPid, containerdPid)
	requireSessionLeader(t, db, containerdShimPid, containerdPid)
	requireEntryLeader(t, db, containerdShimPid, containerdPid, Init)
}

//	/                             (pid, sid, entry meta, entry leader)
//
// systemd
//
//	\___ containerd               (100, 100, "init", 100)
//	|
//	\___ containerd-shim-runc-v2  (1000, 100, "init", 100)
//	      \___ bash               (1001, 1001, "container", 1000)
//
//	Note that containerd originally forks and exec's
//	containerd-shim-runc-v2, which then forks such that it is reparented to
//	init.
func TestInitContainerdShimBashContainerdShimIsReparentedToInit(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	containerdPid := uint32(100)
	containerdShimPid := uint32(1000)
	bashPid := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdPath,
		Pids: types.PidInfo{
			Tgid: containerdPid,
			Sid:  containerdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		Pids: types.PidInfo{
			Tgid: containerdShimPid,
			Sid:  containerdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: containerdShimPid,
		},
	})

	// containerd
	requireProcess(t, db, containerdPid, containerdPath)
	requireParent(t, db, containerdPid, 1)
	requireSessionLeader(t, db, containerdPid, containerdPid)
	requireEntryLeader(t, db, containerdPid, containerdPid, Init)

	// containerd-shim-runc-v2
	requireProcess(t, db, containerdShimPid, containerdShimPath)
	requireParent(t, db, containerdShimPid, 1)
	requireSessionLeader(t, db, containerdShimPid, containerdPid)
	requireEntryLeader(t, db, containerdShimPid, containerdPid, Init)

	// bash
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, containerdShimPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Container)
}

// /                               (pid, sid, entry meta, entry leader)
//
// systemd
//
//	\___ containerd               (100, 100, "init", 100)
//	|
//	\___ containerd-shim-runc-v2  (1000, 100, "init", 100)
//	      \___ pause              (1001, 1001, "container", 1001)
//
// The pause binary is a Kubernetes internal binary that is exec'd in a
// container by the container runtime. It is responsible for holding
// open the pod sandbox while other containers start and stop
func TestInitContainerdShimPauseContainerdShimIsReparentedToInit(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	containerdPid := uint32(100)
	containerdShimPid := uint32(1000)
	pausePid := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdPath,
		Pids: types.PidInfo{
			Tgid: containerdPid,
			Sid:  containerdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		Pids: types.PidInfo{
			Tgid: containerdShimPid,
			Sid:  containerdPid,
			Ppid: 1,
		},
	})

	pausePath := "/usr/bin/pause"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: pausePath,
		Pids: types.PidInfo{
			Tgid: pausePid,
			Sid:  pausePid,
			Ppid: containerdShimPid,
		},
	})

	// containerd
	requireProcess(t, db, containerdPid, containerdPath)
	requireParent(t, db, containerdPid, 1)
	requireSessionLeader(t, db, containerdPid, containerdPid)
	requireEntryLeader(t, db, containerdPid, containerdPid, Init)

	// containerd-shim-runc-v2
	requireProcess(t, db, containerdShimPid, containerdShimPath)
	requireParent(t, db, containerdShimPid, 1)
	requireSessionLeader(t, db, containerdShimPid, containerdPid)
	requireEntryLeader(t, db, containerdShimPid, containerdPid, Init)

	// pause
	requireProcess(t, db, pausePid, pausePath)
	requireParent(t, db, pausePid, containerdShimPid)
	requireSessionLeader(t, db, pausePid, pausePid)
	requireEntryLeader(t, db, pausePid, pausePid, Container)
}

// /                       (pid, sid, entry meta, entry leader)
//
// systemd                 (1, 1, none, none)
//
//	\___ sshd              (100, 100, "init", 100)
//	      \___ bash        (1000, 1000, "sshd", 1000)
//	            \___ ls    (1001, 1000, "sshd", 1000)
//	            |
//	            \___ grep  (1002, 1000, "sshd", 1000) /* ppid/sid data is missing */
//
// Grep does not have ppid or sid set, only pgid. Entry evaluation code
// should fallback to grabbing entry leader data from ls, the process group
// leader.
func TestInitSshdBashLsAndGrepGrepOnlyHasGroupLeader(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshdPid := uint32(100)
	bashPid := uint32(1000)
	lsPid := uint32(1001)
	grepPid := uint32(1002)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshdPid,
			Sid:  sshdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: sshdPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		Pids: types.PidInfo{
			Tgid: lsPid,
			Sid:  bashPid,
			Ppid: bashPid,
			Pgid: lsPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: grepPath,
		Pids: types.PidInfo{
			Tgid: grepPid,
			Pgid: lsPid,
		},
	})

	// sshd
	requireProcess(t, db, sshdPid, sshdPath)
	requireParent(t, db, sshdPid, 1)
	requireSessionLeader(t, db, sshdPid, sshdPid)
	requireEntryLeader(t, db, sshdPid, sshdPid, Init)

	// bash
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, sshdPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Sshd)

	// ls
	requireProcess(t, db, lsPid, lsPath)
	requireParent(t, db, lsPid, bashPid)
	requireSessionLeader(t, db, lsPid, bashPid)
	requireEntryLeader(t, db, lsPid, bashPid, Sshd)

	// grep
	grep, err := db.GetProcess(grepPid)
	require.Nil(t, err)
	requireParentUnset(t, grep)

	requireProcess(t, db, grepPid, grepPath)
	requireEntryLeader(t, db, grepPid, bashPid, Sshd)
}

// /                       (pid, sid, entry meta, entry leader)
//
// systemd                 (1, 1, none, none)
//
//	\___ sshd              (100, 100, "init", 100)
//	      \___ bash        (1000, 1000, "sshd", 1000)
//	            \___ ls    (1001, 1000, "sshd", 1000)
//	            |
//	            \___ grep  (1002, 1000, "sshd", 1000) /* ppid/pgid data is missing */
//
// Grep does not have ppid or pgid set, ppid. Entry evaluation code should
// fallback to grabbing entry leader data from sshd, the session leader.
func TestInitSshdBashLsAndGrepGrepOnlyHasSessionLeader(t *testing.T) {
	reader := procfs.NewMockReader()
	populateProcfsWithInit(reader)
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshdPid := uint32(100)
	bashPid := uint32(1000)
	lsPid := uint32(1001)
	grepPid := uint32(1002)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		Pids: types.PidInfo{
			Tgid: sshdPid,
			Sid:  sshdPid,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		Pids: types.PidInfo{
			Tgid: bashPid,
			Sid:  bashPid,
			Ppid: sshdPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		Pids: types.PidInfo{
			Tgid: lsPid,
			Sid:  bashPid,
			Ppid: bashPid,
			Pgid: lsPid,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: grepPath,
		Pids: types.PidInfo{
			Tgid: grepPid,
			Sid:  bashPid,
		},
	})

	// sshd
	requireProcess(t, db, sshdPid, sshdPath)
	requireParent(t, db, sshdPid, 1)
	requireSessionLeader(t, db, sshdPid, sshdPid)
	requireEntryLeader(t, db, sshdPid, sshdPid, Init)

	// bash
	requireProcess(t, db, bashPid, bashPath)
	requireParent(t, db, bashPid, sshdPid)
	requireSessionLeader(t, db, bashPid, bashPid)
	requireEntryLeader(t, db, bashPid, bashPid, Sshd)

	// ls
	requireProcess(t, db, lsPid, lsPath)
	requireParent(t, db, lsPid, bashPid)
	requireSessionLeader(t, db, lsPid, bashPid)
	requireEntryLeader(t, db, lsPid, bashPid, Sshd)

	// grep
	grep, err := db.GetProcess(grepPid)
	require.Nil(t, err)
	requireParentUnset(t, grep)

	requireProcess(t, db, grepPid, grepPath)
	requireSessionLeader(t, db, grepPid, bashPid)
	requireEntryLeader(t, db, grepPid, bashPid, Sshd)
}

// /     (pid, sid, entry meta, entry leader)
//
// grep  (1001, 1000, "unknown", 1001)
//
// No parent, session leader, or process group leader exists to draw
// on to get an entry leader for grep, fallback to assigning it an
// entry meta type of "unknown" and making it an entry leader.
func TestGrepInIsolation(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	grepPid := uint32(1001)

	db.InsertExec(types.ProcessExecEvent{
		Filename: grepPath,
		Pids: types.PidInfo{
			Tgid: grepPid,
			Ppid: 1000,
			Sid:  grepPid,
		},
	})

	process, err := db.GetProcess(grepPid)
	require.Nil(t, err)
	requireParentUnset(t, process)

	requireProcess(t, db, grepPid, grepPath)
	requireSessionLeader(t, db, grepPid, grepPid)
	requireEntryLeader(t, db, grepPid, grepPid, EntryUnknown)
}

// /                              (pid, sid, entry meta, entry leader)
//
// kthreadd                       (2, 0, <none>, <none>)
//
//	\___ rcu_gp                   (3, 0, <none>, <none>)
//
// Kernel threads should never have an entry meta type or entry leader set.
func TestKernelThreads(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)

	kthreaddPid := uint32(2)
	rcuGpPid := uint32(3)

	kthreaddPath := "kthreadd"
	rcuGpPath := "rcu_gp"

	db.InsertExec(types.ProcessExecEvent{
		Filename: kthreaddPath,
		Pids: types.PidInfo{
			Tgid: kthreaddPid,
			Ppid: 1,
			Sid:  0,
		},
	})

	db.InsertExec(types.ProcessExecEvent{
		Filename: rcuGpPath,
		Pids: types.PidInfo{
			Tgid: rcuGpPid,
			Ppid: kthreaddPid,
			Sid:  0,
		},
	})

	// kthreadd
	kthreadd, err := db.GetProcess(kthreaddPid)
	require.Nil(t, err)
	requireParentUnset(t, kthreadd)
	requireSessionLeaderUnset(t, kthreadd)
	requireEntryLeaderUnset(t, kthreadd)

	requireProcess(t, db, kthreaddPid, kthreaddPath)

	// rcu_gp
	rcuGp, err := db.GetProcess(rcuGpPid)
	require.Nil(t, err)
	requireSessionLeaderUnset(t, rcuGp)
	requireEntryLeaderUnset(t, rcuGp)

	requireProcess(t, db, rcuGpPid, rcuGpPath)
	requireParent(t, db, rcuGpPid, kthreaddPid)
}
