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

func requireEntryLeader(t *testing.T, db *DB, pid uint32, entryPID uint32, expectedEntryType EntryType) {
	t.Helper()
	process, err := db.GetProcess(pid)
	require.Nil(t, err)
	require.Equal(t, entryPID, process.EntryLeader.PID)
	require.NotNil(t, process.EntryLeader.SameAsProcess)
	require.Equal(t, pid == entryPID, *process.EntryLeader.SameAsProcess)

	entryType, err := db.GetEntryType(entryPID)
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
	fork.ChildPIDs = exec.PIDs
	parent, err := db.GetProcess(exec.PIDs.Ppid)
	if err != nil {
		fork.ParentPIDs = exec.PIDs
		fork.ParentPIDs.Tgid = exec.PIDs.Ppid
		fork.ParentPIDs.Ppid = 0
		fork.ParentPIDs.Pgid = 0

		fork.ChildPIDs.Pgid = exec.PIDs.Ppid

		// if the exec makes itself a session and the parent is no where to be
		// found we'll make the parent its own session
		if exec.PIDs.Tgid == exec.PIDs.Sid {
			fork.ParentPIDs.Sid = exec.PIDs.Ppid
		}
	} else {
		fork.ParentPIDs.Tgid = parent.PID
		fork.ParentPIDs.Ppid = parent.Parent.PID
		fork.ParentPIDs.Sid = parent.SessionLeader.PID

		// keep group leader the same for now
		fork.ParentPIDs.Pgid = exec.PIDs.Pgid
	}

	if fork.ParentPIDs.Tgid != 0 {
		db.InsertFork(fork)
	}

	db.InsertExec(exec)
}

var systemdPath = "/sbin/systemd"

func populateProcfsWithInit(reader *procfs.MockReader) {
	reader.AddEntry(1, procfs.ProcessInfo{
		PIDs: types.PIDInfo{
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
		PIDs: types.PIDInfo{
			Tgid: pid,
			Sid:  pid,
		},
		CTTY: types.TTYDev{
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
		PIDs: types.PIDInfo{
			Tgid: pid,
			Sid:  pid,
		},
		CTTY: types.TTYDev{
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
		PIDs: types.PIDInfo{
			Tgid: pid,
			Sid:  pid,
			Ppid: 1,
		},
		CTTY: types.TTYDev{
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

	ssmPID := uint32(999)
	bashPID := uint32(1000)
	ssmPath := "/usr/bin/ssm-session-worker"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: ssmPath,
		PIDs: types.PIDInfo{
			Tgid: ssmPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: ssmPID,
		},
		CTTY: types.TTYDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, ssmPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Ssm)
}

func TestSingleProcessSessionLeaderChildOfSshd(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	sshdPID := uint32(999)
	bashPID := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshdPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: sshdPID,
		},
		CTTY: types.TTYDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, sshdPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Sshd)
}

func TestSingleProcessSessionLeaderChildOfContainerdShim(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	containerdShimPID := uint32(999)
	bashPID := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		PIDs: types.PIDInfo{
			Tgid: containerdShimPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: containerdShimPID,
		},
		CTTY: types.TTYDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, containerdShimPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Container)
}

func TestSingleProcessSessionLeaderChildOfRunc(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	runcPID := uint32(999)
	bashPID := uint32(1000)
	runcPath := "/bin/runc"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: runcPath,
		PIDs: types.PIDInfo{
			Tgid: runcPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: runcPID,
		},
		CTTY: types.TTYDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, runcPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Container)
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
		PIDs: types.PIDInfo{
			Tgid: pid,
			Sid:  pid,
		},
		CTTY: types.TTYDev{
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

// Entry evaluation code should overwrite an old EntryLeaderPID and
// EntryLeaderEntryMetaType
func TestSingleProcessOverwriteOldEntryLeader(t *testing.T) {
	reader := procfs.NewMockReader()
	db := NewDB(reader, *logger)
	db.ScrapeProcfs()

	ssmPID := uint32(999)
	bashPID := uint32(1000)
	ssmPath := "/usr/bin/ssm-session-worker"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: ssmPath,
		PIDs: types.PIDInfo{
			Tgid: ssmPID,
			Sid:  ssmPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  ssmPID,
			Ppid: ssmPID,
		},
		CTTY: types.TTYDev{
			Major: 136,
			Minor: 62,
		},
	})

	// bash is not a session leader so it shouldn't be an entry leader. Its
	// entry leader should be ssm, which is an init entry leader
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, ssmPID)
	requireSessionLeader(t, db, bashPID, ssmPID)
	requireEntryLeader(t, db, bashPID, ssmPID, Init)

	// skiping setsid event and assuming the pids will be updated in this exec
	db.InsertExec(types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: ssmPID,
		},
		CTTY: types.TTYDev{
			Major: 136,
			Minor: 62,
		},
	})

	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, ssmPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Ssm)
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

	sshdPID := uint32(100)
	bashPID := uint32(1000)
	lsPID := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshdPID,
			Sid:  sshdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: sshdPID,
			Pgid: bashPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		PIDs: types.PIDInfo{
			Tgid: lsPID,
			Sid:  bashPID,
			Ppid: bashPID,
			Pgid: lsPID,
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
	requireProcess(t, db, sshdPID, sshdPath)
	requireParent(t, db, sshdPID, 1)
	requireSessionLeader(t, db, sshdPID, sshdPID)
	requireEntryLeader(t, db, sshdPID, sshdPID, Init)

	// bash
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, sshdPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Sshd)
	requireGroupLeader(t, db, bashPID, bashPID)

	// ls
	requireProcess(t, db, lsPID, lsPath)
	requireParent(t, db, lsPID, bashPID)
	requireSessionLeader(t, db, lsPID, bashPID)
	requireEntryLeader(t, db, lsPID, bashPID, Sshd)
	requireGroupLeader(t, db, lsPID, lsPID)
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

	sshd0PID := uint32(100)
	sshd1PID := uint32(101)
	bashPID := uint32(1000)
	lsPID := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshd0PID,
			Sid:  sshd0PID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshd1PID,
			Sid:  sshd1PID,
			Ppid: sshd0PID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: sshd1PID,
			Pgid: bashPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		PIDs: types.PIDInfo{
			Tgid: lsPID,
			Sid:  bashPID,
			Ppid: bashPID,
			Pgid: lsPID,
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
	requireProcess(t, db, sshd0PID, sshdPath)
	requireParent(t, db, sshd0PID, 1)
	requireSessionLeader(t, db, sshd0PID, sshd0PID)
	requireEntryLeader(t, db, sshd0PID, sshd0PID, Init)

	// sshd1
	requireProcess(t, db, sshd1PID, sshdPath)
	requireParent(t, db, sshd1PID, sshd0PID)
	requireSessionLeader(t, db, sshd1PID, sshd1PID)
	requireEntryLeader(t, db, sshd1PID, sshd0PID, Init)

	// bash
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, sshd1PID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Sshd)

	// ls
	requireProcess(t, db, lsPID, lsPath)
	requireParent(t, db, lsPID, bashPID)
	requireSessionLeader(t, db, lsPID, bashPID)
	requireEntryLeader(t, db, lsPID, bashPID, Sshd)
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

	sshd0PID := uint32(100)
	sshd1PID := uint32(101)
	sshd2PID := uint32(102)
	bashPID := uint32(1000)
	lsPID := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshd0PID,
			Sid:  sshd0PID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshd1PID,
			Sid:  sshd1PID,
			Ppid: sshd0PID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshd2PID,
			Sid:  sshd1PID,
			Ppid: sshd1PID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: sshd2PID,
			Pgid: bashPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		PIDs: types.PIDInfo{
			Tgid: lsPID,
			Sid:  bashPID,
			Ppid: bashPID,
			Pgid: lsPID,
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
	requireProcess(t, db, sshd0PID, sshdPath)
	requireParent(t, db, sshd0PID, 1)
	requireSessionLeader(t, db, sshd0PID, sshd0PID)
	requireEntryLeader(t, db, sshd0PID, sshd0PID, Init)

	// sshd1
	requireProcess(t, db, sshd1PID, sshdPath)
	requireParent(t, db, sshd1PID, sshd0PID)
	requireSessionLeader(t, db, sshd1PID, sshd1PID)
	requireEntryLeader(t, db, sshd1PID, sshd0PID, Init)

	// sshd2
	requireProcess(t, db, sshd2PID, sshdPath)
	requireParent(t, db, sshd2PID, sshd1PID)
	requireSessionLeader(t, db, sshd2PID, sshd1PID)
	requireEntryLeader(t, db, sshd2PID, sshd0PID, Init)

	// bash
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, sshd2PID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Sshd)

	// ls
	requireProcess(t, db, lsPID, lsPath)
	requireParent(t, db, lsPID, bashPID)
	requireSessionLeader(t, db, lsPID, bashPID)
	requireEntryLeader(t, db, lsPID, bashPID, Sshd)
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

	containerdPID := uint32(100)
	containerdShimPID := uint32(1000)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdPath,
		PIDs: types.PIDInfo{
			Tgid: containerdPID,
			Sid:  containerdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		PIDs: types.PIDInfo{
			Tgid: containerdShimPID,
			Sid:  containerdPID,
			Ppid: containerdPID,
		},
	})

	// containerd
	requireProcess(t, db, containerdPID, containerdPath)
	requireParent(t, db, containerdPID, 1)
	requireSessionLeader(t, db, containerdPID, containerdPID)
	requireEntryLeader(t, db, containerdPID, containerdPID, Init)

	// containerd-shim-runc-v2
	requireProcess(t, db, containerdShimPID, containerdShimPath)
	requireParent(t, db, containerdShimPID, containerdPID)
	requireSessionLeader(t, db, containerdShimPID, containerdPID)
	requireEntryLeader(t, db, containerdShimPID, containerdPID, Init)
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

	containerdPID := uint32(100)
	containerdShimPID := uint32(1000)
	bashPID := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdPath,
		PIDs: types.PIDInfo{
			Tgid: containerdPID,
			Sid:  containerdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		PIDs: types.PIDInfo{
			Tgid: containerdShimPID,
			Sid:  containerdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: containerdShimPID,
		},
	})

	// containerd
	requireProcess(t, db, containerdPID, containerdPath)
	requireParent(t, db, containerdPID, 1)
	requireSessionLeader(t, db, containerdPID, containerdPID)
	requireEntryLeader(t, db, containerdPID, containerdPID, Init)

	// containerd-shim-runc-v2
	requireProcess(t, db, containerdShimPID, containerdShimPath)
	requireParent(t, db, containerdShimPID, 1)
	requireSessionLeader(t, db, containerdShimPID, containerdPID)
	requireEntryLeader(t, db, containerdShimPID, containerdPID, Init)

	// bash
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, containerdShimPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Container)
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

	containerdPID := uint32(100)
	containerdShimPID := uint32(1000)
	pausePID := uint32(1001)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdPath,
		PIDs: types.PIDInfo{
			Tgid: containerdPID,
			Sid:  containerdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: containerdShimPath,
		PIDs: types.PIDInfo{
			Tgid: containerdShimPID,
			Sid:  containerdPID,
			Ppid: 1,
		},
	})

	pausePath := "/usr/bin/pause"

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: pausePath,
		PIDs: types.PIDInfo{
			Tgid: pausePID,
			Sid:  pausePID,
			Ppid: containerdShimPID,
		},
	})

	// containerd
	requireProcess(t, db, containerdPID, containerdPath)
	requireParent(t, db, containerdPID, 1)
	requireSessionLeader(t, db, containerdPID, containerdPID)
	requireEntryLeader(t, db, containerdPID, containerdPID, Init)

	// containerd-shim-runc-v2
	requireProcess(t, db, containerdShimPID, containerdShimPath)
	requireParent(t, db, containerdShimPID, 1)
	requireSessionLeader(t, db, containerdShimPID, containerdPID)
	requireEntryLeader(t, db, containerdShimPID, containerdPID, Init)

	// pause
	requireProcess(t, db, pausePID, pausePath)
	requireParent(t, db, pausePID, containerdShimPID)
	requireSessionLeader(t, db, pausePID, pausePID)
	requireEntryLeader(t, db, pausePID, pausePID, Container)
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

	sshdPID := uint32(100)
	bashPID := uint32(1000)
	lsPID := uint32(1001)
	grepPID := uint32(1002)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshdPID,
			Sid:  sshdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: sshdPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		PIDs: types.PIDInfo{
			Tgid: lsPID,
			Sid:  bashPID,
			Ppid: bashPID,
			Pgid: lsPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: grepPath,
		PIDs: types.PIDInfo{
			Tgid: grepPID,
			Pgid: lsPID,
		},
	})

	// sshd
	requireProcess(t, db, sshdPID, sshdPath)
	requireParent(t, db, sshdPID, 1)
	requireSessionLeader(t, db, sshdPID, sshdPID)
	requireEntryLeader(t, db, sshdPID, sshdPID, Init)

	// bash
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, sshdPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Sshd)

	// ls
	requireProcess(t, db, lsPID, lsPath)
	requireParent(t, db, lsPID, bashPID)
	requireSessionLeader(t, db, lsPID, bashPID)
	requireEntryLeader(t, db, lsPID, bashPID, Sshd)

	// grep
	grep, err := db.GetProcess(grepPID)
	require.Nil(t, err)
	requireParentUnset(t, grep)

	requireProcess(t, db, grepPID, grepPath)
	requireEntryLeader(t, db, grepPID, bashPID, Sshd)
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

	sshdPID := uint32(100)
	bashPID := uint32(1000)
	lsPID := uint32(1001)
	grepPID := uint32(1002)

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: sshdPath,
		PIDs: types.PIDInfo{
			Tgid: sshdPID,
			Sid:  sshdPID,
			Ppid: 1,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: bashPath,
		PIDs: types.PIDInfo{
			Tgid: bashPID,
			Sid:  bashPID,
			Ppid: sshdPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: lsPath,
		PIDs: types.PIDInfo{
			Tgid: lsPID,
			Sid:  bashPID,
			Ppid: bashPID,
			Pgid: lsPID,
		},
	})

	insertForkAndExec(t, db, types.ProcessExecEvent{
		Filename: grepPath,
		PIDs: types.PIDInfo{
			Tgid: grepPID,
			Sid:  bashPID,
		},
	})

	// sshd
	requireProcess(t, db, sshdPID, sshdPath)
	requireParent(t, db, sshdPID, 1)
	requireSessionLeader(t, db, sshdPID, sshdPID)
	requireEntryLeader(t, db, sshdPID, sshdPID, Init)

	// bash
	requireProcess(t, db, bashPID, bashPath)
	requireParent(t, db, bashPID, sshdPID)
	requireSessionLeader(t, db, bashPID, bashPID)
	requireEntryLeader(t, db, bashPID, bashPID, Sshd)

	// ls
	requireProcess(t, db, lsPID, lsPath)
	requireParent(t, db, lsPID, bashPID)
	requireSessionLeader(t, db, lsPID, bashPID)
	requireEntryLeader(t, db, lsPID, bashPID, Sshd)

	// grep
	grep, err := db.GetProcess(grepPID)
	require.Nil(t, err)
	requireParentUnset(t, grep)

	requireProcess(t, db, grepPID, grepPath)
	requireSessionLeader(t, db, grepPID, bashPID)
	requireEntryLeader(t, db, grepPID, bashPID, Sshd)
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

	grepPID := uint32(1001)

	db.InsertExec(types.ProcessExecEvent{
		Filename: grepPath,
		PIDs: types.PIDInfo{
			Tgid: grepPID,
			Ppid: 1000,
			Sid:  grepPID,
		},
	})

	process, err := db.GetProcess(grepPID)
	require.Nil(t, err)
	requireParentUnset(t, process)

	requireProcess(t, db, grepPID, grepPath)
	requireSessionLeader(t, db, grepPID, grepPID)
	requireEntryLeader(t, db, grepPID, grepPID, EntryUnknown)
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

	kthreaddPID := uint32(2)
	rcuGpPID := uint32(3)

	kthreaddPath := "kthreadd"
	rcuGpPath := "rcu_gp"

	db.InsertExec(types.ProcessExecEvent{
		Filename: kthreaddPath,
		PIDs: types.PIDInfo{
			Tgid: kthreaddPID,
			Ppid: 1,
			Sid:  0,
		},
	})

	db.InsertExec(types.ProcessExecEvent{
		Filename: rcuGpPath,
		PIDs: types.PIDInfo{
			Tgid: rcuGpPID,
			Ppid: kthreaddPID,
			Sid:  0,
		},
	})

	// kthreadd
	kthreadd, err := db.GetProcess(kthreaddPID)
	require.Nil(t, err)
	requireParentUnset(t, kthreadd)
	requireSessionLeaderUnset(t, kthreadd)
	requireEntryLeaderUnset(t, kthreadd)

	requireProcess(t, db, kthreaddPID, kthreaddPath)

	// rcu_gp
	rcuGp, err := db.GetProcess(rcuGpPID)
	require.Nil(t, err)
	requireSessionLeaderUnset(t, rcuGp)
	requireEntryLeaderUnset(t, rcuGp)

	requireProcess(t, db, rcuGpPID, rcuGpPath)
	requireParent(t, db, rcuGpPID, kthreaddPID)
}
