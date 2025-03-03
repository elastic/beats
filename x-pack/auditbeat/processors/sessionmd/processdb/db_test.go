// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/auditbeat/helper/tty"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/timeutils"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var logger = logp.NewLogger("processdb")

var testAlwaysTimeout = func(_, _ time.Time) bool {
	return false
}

var testNeverTimeout = func(_, _ time.Time) bool {
	return true
}

func TestGetTTYType(t *testing.T) {
	require.Equal(t, tty.TTYConsole, tty.GetTTYType(4, 0))
	require.Equal(t, tty.Pts, tty.GetTTYType(136, 0))
	require.Equal(t, tty.TTY, tty.GetTTYType(4, 64))
	require.Equal(t, tty.TTYUnknown, tty.GetTTYType(1000, 1000))
}

func TestProcessOrphanResolve(t *testing.T) {
	// test to make sure that if we get an exit event before a exec event, we still match up the two

	// uncomment if you want some logs
	//_ = logp.DevelopmentSetup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := procfs.NewProcfsReader(*logger)
	testDB, err := NewDB(ctx, monitoring.NewRegistry(), reader, logp.L(), -1, false)
	require.NoError(t, err)
	removalFuncTimeoutWaiting = testAlwaysTimeout

	pid1 := types.PIDInfo{Tgid: 10, StartTimeNS: 19}
	pid2 := types.PIDInfo{Tgid: 11, StartTimeNS: 25}

	exitCode1 := int32(24)
	exitCode2 := int32(30)

	exit1 := types.ProcessExitEvent{PIDs: pid1, ExitCode: exitCode1}
	exit2 := types.ProcessExitEvent{PIDs: pid2, ExitCode: exitCode2}

	exec1 := types.ProcessExecEvent{PIDs: pid1}
	exec2 := types.ProcessExecEvent{PIDs: pid2}

	testDB.InsertExit(exit1)
	testDB.InsertExit(exit2)

	testDB.InsertExec(exec1)
	testDB.InsertExec(exec2)

	res1, err := testDB.GetProcess(pid1.Tgid)
	require.NoError(t, err)
	require.Equal(t, exitCode1, res1.ExitCode)
	require.Equal(t, timeutils.TimeFromNsSinceBoot(timeutils.ReduceTimestampPrecision(pid1.StartTimeNS)), res1.Start)

	res2, err := testDB.GetProcess(pid2.Tgid)
	require.NoError(t, err)
	require.Equal(t, exitCode2, res2.ExitCode)
	require.Equal(t, timeutils.TimeFromNsSinceBoot(timeutils.ReduceTimestampPrecision(pid2.StartTimeNS)), res2.Start)
	// verify that the pid is removed once we run a pass of the reaper
	require.Len(t, testDB.processes, 2)
	require.Len(t, testDB.removalMap, 2)
	testDB.reapProcs()
	require.Len(t, testDB.processes, 0)
	require.Len(t, testDB.removalMap, 0)
}

func TestReapExitOrphans(t *testing.T) {
	// test to make sure that orphaned exit events are still cleaned up
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := procfs.NewProcfsReader(*logger)
	testDB, err := NewDB(ctx, monitoring.NewRegistry(), reader, logp.L(), -1, false)
	require.NoError(t, err)
	removalFuncTimeoutWaiting = testAlwaysTimeout
	orphanFuncTimeoutWaiting = testAlwaysTimeout

	testDB.InsertExit(types.ProcessExitEvent{PIDs: types.PIDInfo{Tgid: 10, StartTimeNS: 19}, ExitCode: 0})
	testDB.InsertExit(types.ProcessExitEvent{PIDs: types.PIDInfo{Tgid: 11, StartTimeNS: 20}, ExitCode: 0})
	testDB.InsertExit(types.ProcessExitEvent{PIDs: types.PIDInfo{Tgid: 12, StartTimeNS: 25}, ExitCode: 0})

	require.Len(t, testDB.removalMap, 3)

	testDB.reapProcs()

	require.Len(t, testDB.removalMap, 0)
}

func TestReapProcesses(t *testing.T) {
	reader := procfs.NewProcfsReader(*logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testDB, err := NewDB(ctx, monitoring.NewRegistry(), reader, logp.L(), -1, true)
	require.NoError(t, err)
	testDB.processReapAfter = time.Duration(0)
	removalFuncTimeoutWaiting = testNeverTimeout

	pid1 := types.PIDInfo{Tgid: 10, StartTimeNS: 19}
	pid2 := types.PIDInfo{Tgid: 11, StartTimeNS: 25}
	pid3 := types.PIDInfo{Tgid: 13, StartTimeNS: 40}
	pid4 := types.PIDInfo{Tgid: 14, StartTimeNS: 50}

	exec1 := types.ProcessExecEvent{PIDs: pid1, ProcfsLookupFail: true}
	exec2 := types.ProcessExecEvent{PIDs: pid2, ProcfsLookupFail: true}
	exec3 := types.ProcessExecEvent{PIDs: pid3, ProcfsLookupFail: true}
	// if we got a procfs lookup, don't reap
	exec4 := types.ProcessExecEvent{PIDs: pid4, ProcfsLookupFail: false}

	testDB.InsertExec(exec1)
	testDB.InsertExec(exec2)
	testDB.InsertExec(exec3)
	testDB.InsertExec(exec4)

	// if a process has a corresponding exit, do not reap
	testDB.InsertExit(types.ProcessExitEvent{PIDs: pid3, ExitCode: 0})

	testDB.reapProcs()

	// make sure processes are removed
	require.NotContains(t, testDB.processes, pid1.Tgid)
	require.NotContains(t, testDB.processes, pid2.Tgid)
	require.Contains(t, testDB.processes, pid3.Tgid)
	require.Contains(t, testDB.processes, pid4.Tgid)
}

func TestReapProcessesWithProcFS(t *testing.T) {
	mockReader := procfs.NewMockReader()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testDB, err := NewDB(ctx, monitoring.NewRegistry(), mockReader, logp.L(), -1, false)
	require.NoError(t, err)
	testDB.reapProcesses = true
	testDB.processReapAfter = time.Duration(0)
	removalFuncTimeoutWaiting = testNeverTimeout
	orphanFuncTimeoutWaiting = testAlwaysTimeout

	// insert procfs entries for two of the pids
	mockReader.AddEntry(10, procfs.ProcessInfo{})
	mockReader.AddEntry(11, procfs.ProcessInfo{})

	pid1 := types.PIDInfo{Tgid: 10, StartTimeNS: 19}
	pid2 := types.PIDInfo{Tgid: 11, StartTimeNS: 25}
	pid3 := types.PIDInfo{Tgid: 13, StartTimeNS: 40}

	exec1 := types.ProcessExecEvent{PIDs: pid1, ProcfsLookupFail: false}
	exec2 := types.ProcessExecEvent{PIDs: pid2, ProcfsLookupFail: false}
	exec3 := types.ProcessExecEvent{PIDs: pid3, ProcfsLookupFail: false}

	testDB.InsertExec(exec1)
	testDB.InsertExec(exec2)
	testDB.InsertExec(exec3)

	testDB.reapProcs()
	// after one iteration, 3 should be marked as `LookupFail`, others should be fine
	require.True(t, testDB.processes[pid3.Tgid].procfsLookupFail)
	require.False(t, testDB.processes[pid2.Tgid].procfsLookupFail)
	require.False(t, testDB.processes[pid1.Tgid].procfsLookupFail)

	// after a second reap, they should be removed
	testDB.reapProcs()

	require.NotContains(t, testDB.processes, pid3.Tgid)
	require.Contains(t, testDB.processes, pid1.Tgid)
	require.Contains(t, testDB.processes, pid2.Tgid)
}

func TestReapingProcessesOrphanResolvedRace(t *testing.T) {
	// test to make sure that if we resolve a process in between mutex holds, we won't prematurely reap it
	mockReader := procfs.NewMockReader()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testDB, err := NewDB(ctx, monitoring.NewRegistry(), mockReader, logp.L(), -1, false)
	require.NoError(t, err)
	testDB.reapProcesses = true
	testDB.processReapAfter = time.Duration(0)
	removalFuncTimeoutWaiting = testNeverTimeout
	orphanFuncTimeoutWaiting = testAlwaysTimeout

	// insert procfs entries for two of the pids
	pid1 := types.PIDInfo{Tgid: 10, StartTimeNS: 19}
	exec1 := types.ProcessExecEvent{PIDs: pid1, ProcfsLookupFail: false}
	testDB.InsertExec(exec1)

	testDB.reapProcs()
	// should now be marked as lookup fail
	require.True(t, testDB.processes[pid1.Tgid].procfsLookupFail)

	// now we get our exit
	testDB.InsertExit(types.ProcessExitEvent{PIDs: pid1})
	testDB.reapProcs()
	// process should still exist
	require.Contains(t, testDB.processes, pid1.Tgid)
}
