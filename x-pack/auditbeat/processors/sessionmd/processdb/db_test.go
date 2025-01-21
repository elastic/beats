// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/auditbeat/helper/tty"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/timeutils"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
)

var logger = logp.NewLogger("processdb")

var testAlwaysTimeout = func(_, _ time.Time) bool {
	return false
}

func TestGetTTYType(t *testing.T) {
	require.Equal(t, tty.TTYConsole, tty.GetTTYType(4, 0))
	require.Equal(t, tty.Pts, tty.GetTTYType(136, 0))
	require.Equal(t, tty.TTY, tty.GetTTYType(4, 64))
	require.Equal(t, tty.TTYUnknown, tty.GetTTYType(1000, 1000))
}
func TestProcessOrphanResolve(t *testing.T) {
	//test to make sure that if we get an exit event before a exec event, we still match up the two

	// uncomment if you want some logs
	//_ = logp.DevelopmentSetup()
	reader := procfs.NewProcfsReader(*logger)
	testDB, err := NewDB(reader, *logp.L(), time.Minute)
	require.NoError(t, err)
	testDB.skipReaper = true
	functionTimeoutReached = testAlwaysTimeout

	pid1 := types.PIDInfo{
		Tgid:        10,
		StartTimeNS: 19,
	}
	pid2 := types.PIDInfo{
		Tgid:        11,
		StartTimeNS: 25,
	}

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
	//test to make sure that orphaned exit events are still cleaned up

	reader := procfs.NewProcfsReader(*logger)
	testDB, err := NewDB(reader, *logp.L(), time.Minute)
	require.NoError(t, err)
	testDB.skipReaper = true
	functionTimeoutReached = testAlwaysTimeout

	testDB.InsertExit(types.ProcessExitEvent{PIDs: types.PIDInfo{Tgid: 10, StartTimeNS: 19}, ExitCode: 0})
	testDB.InsertExit(types.ProcessExitEvent{PIDs: types.PIDInfo{Tgid: 11, StartTimeNS: 20}, ExitCode: 0})
	testDB.InsertExit(types.ProcessExitEvent{PIDs: types.PIDInfo{Tgid: 12, StartTimeNS: 25}, ExitCode: 0})

	require.Len(t, testDB.removalMap, 3)

	/// run four times, to pass over the iterations needed for the reaper
	testDB.reapProcs()
	testDB.reapProcs()
	testDB.reapProcs()

	require.Len(t, testDB.removalMap, 0)
}
