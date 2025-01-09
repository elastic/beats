// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (amd64 || arm64) && cgo

package process

import (
	"os"
	"os/exec"
	"testing"
	"time"

	quark "github.com/elastic/go-quark"
	"github.com/stretchr/testify/require"
)

func TestInitialSnapshot(t *testing.T) {
	qq := openQueue(t)
	defer qq.Close()

	// There should be events of kind quark.QUARK_EV_SNAPSHOT
	qevs := drainFor(t, qq, 5*time.Millisecond)
	var gotsnap bool
	for _, qev := range qevs {
		if qev.Events&quark.QUARK_EV_SNAPSHOT != 0 {
			gotsnap = true
		}
	}

	require.True(t, gotsnap)
}

func TestForkExecExit(t *testing.T) {
	qq := openQueue(t)
	defer qq.Close()

	// runNop will fork+exec+exit /bin/true
	cmd := runNop(t)
	qev := drainFirstOfPid(t, qq, cmd.Process.Pid)

	// We should get at least FORK|EXEC|EXIT in the aggregation
	require.Equal(t,
		qev.Events&(quark.QUARK_EV_FORK|quark.QUARK_EV_EXEC|quark.QUARK_EV_EXIT),
		quark.QUARK_EV_FORK|quark.QUARK_EV_EXEC|quark.QUARK_EV_EXIT)

	// This is virtually impossible to fail, but we're pedantic
	require.True(t, qev.Process.Proc.Valid)

	// We need these otherwise nothing works
	require.NotZero(t, qev.Process.Proc.MntInonum)
	require.NotZero(t, qev.Process.Proc.TimeBoot)
	require.NotZero(t, qev.Process.Proc.Ppid)

	// Must be /bin/true
	require.Equal(t, qev.Process.Filename, cmd.Path)
	require.Equal(t, qev.Process.Filename, cmd.Args[0])

	// Check Cwd
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, qev.Process.Cwd, cwd)

	// Check exit
	require.True(t, qev.Process.Exit.Valid)
	require.Zero(t, qev.Process.Exit.ExitCode)
	// Don't care about ExitTime, it's also not precise
}

func openQueue(t *testing.T) *quark.Queue {
	attr := quark.DefaultQueueAttr()
	attr.HoldTime = 25
	qq, err := quark.OpenQueue(attr, 1)
	require.NoError(t, err)

	return qq
}

// runNop does fork+exec+exit /bin/true
func runNop(t *testing.T) *exec.Cmd {
	//	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	cmd := exec.Command("/bin/true")
	require.NotNil(t, cmd)
	err := cmd.Run()
	require.NoError(t, err)

	return cmd
}

// drainFor drains all events for `d`
func drainFor(t *testing.T, qq *quark.Queue, d time.Duration) []quark.Event {
	var allQevs []quark.Event

	start := time.Now()

	for {
		qevs, err := qq.GetEvents()
		require.NoError(t, err)
		for _, qev := range qevs {
			if !wantedEvent(qev) {
				continue
			}
			allQevs = append(allQevs, qev)
		}
		if time.Since(start) > d {
			break
		}
		// Intentionally placed at the end so that we always
		// get one more try after the last block
		if len(qevs) == 0 {
			qq.Block()
		}
	}

	return allQevs
}

// drainFirstOfPid returns the first event
func drainFirstOfPid(t *testing.T, qq *quark.Queue, pid int) quark.Event {
	start := time.Now()

	for {
		qevs, err := qq.GetEvents()
		require.NoError(t, err)
		for _, qev := range qevs {
			if !wantedEvent(qev) {
				continue
			}
			if qev.Process.Pid == uint32(pid) {
				return qev
			}
		}
		if time.Since(start) > time.Second {
			break
		}
		// Intentionally placed at the end so that we always
		// get one more try after the last block
		if len(qevs) == 0 {
			qq.Block()
		}
	}

	t.Fatalf("Can't find event of pid %d", pid)

	return quark.Event{} // NOTREACHED
}

// func filterPid(qevs []quark.Event, pid int) []quark.Event {
// 	var pidQevs []quark.Event

// 	for _, qev := range qevs {
// 		if qev.Process.Pid == uint32(pid) {
// 			pidQevs = append(pidQevs, qev)
// 		}
// 	}

// 	return pidQevs
// }
