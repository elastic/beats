// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package procfs_provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	logger    = *logp.NewLogger("procfs_test")
	timestamp = time.Now()
)

func TestExecveEvent(t *testing.T) {
	var pid uint32 = 100
	event := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			"auditd": mapstr.M{
				"data": mapstr.M{
					"a0":      "aaaad2e476e0",
					"a1":      "aaaad2dd07a0",
					"a2":      "aaaad3170490",
					"a3":      "ffff85911b40",
					"arch":    "aarch64",
					"argc":    "1",
					"syscall": "execve",
					"tty":     "pts4",
				},
			},
			"process": mapstr.M{
				"pid":               100,
				"args":              "whoami",
				"executable":        "/usr/bin/whoami",
				"name":              "whoami",
				"working_directory": "/",
			},
		},
	}
	prereq := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         60,
				Tgid:        60,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         0,
			},
		},
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         80,
				Tgid:        80,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         0,
			},
		},
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         90,
				Tgid:        90,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         0,
			},
		},
	}
	procinfo := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         100,
				Tgid:        100,
				Vpid:        0,
				Ppid:        80,
				Pgid:        90,
				Sid:         60,
			},
		},
	}
	expected := procfs.ProcessInfo{
		PIDs: types.PIDInfo{
			Tgid: 100,
			Ppid: 80,
			Pgid: 90,
			Sid:  60,
		},
	}

	reader := procfs.NewMockReader()
	db, err := processdb.NewDB(reader, logger)
	require.Nil(t, err)
	for _, entry := range prereq {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}
	db.ScrapeProcfs()

	for _, entry := range procinfo {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}

	provider, err := NewProvider(context.TODO(), &logger, db, reader, "process.pid")
	require.Nil(t, err, "error creating provider")

	err = provider.SyncDB(&event, expected.PIDs.Tgid)
	require.Nil(t, err)

	actual, err := db.GetProcess(pid)
	require.Nil(t, err, "pid not found in db")

	require.Equal(t, expected.PIDs.Tgid, actual.PID)
	require.Equal(t, expected.PIDs.Ppid, actual.Parent.PID)
	require.Equal(t, expected.PIDs.Pgid, actual.GroupLeader.PID)
	require.Equal(t, expected.PIDs.Sid, actual.SessionLeader.PID)
}

func TestExecveatEvent(t *testing.T) {
	var pid uint32 = 100
	event := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			"auditd": mapstr.M{
				"data": mapstr.M{
					"a0":      "aaaad2e476e0",
					"a1":      "aaaad2dd07a0",
					"a2":      "aaaad3170490",
					"a3":      "ffff85911b40",
					"arch":    "aarch64",
					"argc":    "1",
					"syscall": "execveat",
					"tty":     "pts4",
				},
			},
			"process": mapstr.M{
				"pid":               100,
				"args":              "whoami",
				"executable":        "/usr/bin/whoami",
				"name":              "whoami",
				"working_directory": "/",
			},
		},
	}
	prereq := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         60,
				Tgid:        60,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         0,
			},
		},
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         80,
				Tgid:        80,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         0,
			},
		},
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         90,
				Tgid:        90,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         0,
			},
		},
	}
	procinfo := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         100,
				Tgid:        100,
				Vpid:        0,
				Ppid:        80,
				Pgid:        90,
				Sid:         60,
			},
		},
	}
	expected := procfs.ProcessInfo{
		PIDs: types.PIDInfo{
			Tgid: 100,
			Ppid: 80,
			Pgid: 90,
			Sid:  60,
		},
	}

	reader := procfs.NewMockReader()
	db, err := processdb.NewDB(reader, logger)
	require.Nil(t, err)
	for _, entry := range prereq {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}
	db.ScrapeProcfs()

	for _, entry := range procinfo {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}

	provider, err := NewProvider(context.TODO(), &logger, db, reader, "process.pid")
	require.Nil(t, err, "error creating provider")

	err = provider.SyncDB(&event, expected.PIDs.Tgid)
	require.Nil(t, err)

	actual, err := db.GetProcess(pid)
	require.Nil(t, err, "pid not found in db")

	require.Equal(t, expected.PIDs.Tgid, actual.PID)
	require.Equal(t, expected.PIDs.Ppid, actual.Parent.PID)
	require.Equal(t, expected.PIDs.Pgid, actual.GroupLeader.PID)
	require.Equal(t, expected.PIDs.Sid, actual.SessionLeader.PID)
}

func TestSetSidEvent(t *testing.T) {
	var pid uint32 = 200
	event := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			"auditd": mapstr.M{
				"data": mapstr.M{
					"a0":      "1",
					"a1":      "ffffeb535e38",
					"a2":      "ffffeb535e48",
					"a3":      "410134",
					"arch":    "aarch64",
					"exit":    "200",
					"syscall": "setsid",
					"tty":     "pts4",
				},
				"result": "success",
			},
			"process": mapstr.M{
				"pid": 200,
				"parent": mapstr.M{
					"pid": 100,
				},
			},
		},
	}
	prereq := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         100,
				Tgid:        100,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         1,
			},
		},
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         200,
				Tgid:        200,
				Vpid:        0,
				Ppid:        100,
				Pgid:        0,
				Sid:         100,
			},
		},
	}
	expected := procfs.ProcessInfo{
		PIDs: types.PIDInfo{
			Tid:  200,
			Tgid: 200,
			Ppid: 100,
			Pgid: 0,
			Sid:  200,
		},
	}

	reader := procfs.NewMockReader()
	db, err := processdb.NewDB(reader, logger)
	require.Nil(t, err)
	for _, entry := range prereq {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}
	db.ScrapeProcfs()

	provider, err := NewProvider(context.TODO(), &logger, db, reader, "process.pid")
	require.Nil(t, err, "error creating provider")

	err = provider.SyncDB(&event, expected.PIDs.Tgid)
	require.Nil(t, err)

	actual, err := db.GetProcess(pid)
	if err != nil {
		require.Fail(t, "pid not found in db")
	}

	require.Equal(t, expected.PIDs.Sid, actual.SessionLeader.PID)
}

func TestSetSidEventFailed(t *testing.T) {
	var pid uint32 = 200
	event := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			"auditd": mapstr.M{
				"data": mapstr.M{
					"a0":      "1",
					"a1":      "ffffefbfcb78",
					"a2":      "ffffefbfcb88",
					"a3":      "410134",
					"arch":    "aarch64",
					"exit":    "EPERM",
					"syscall": "setsid",
					"tty":     "pts4",
				},
				"result": "fail",
			},
			"process": mapstr.M{
				"pid": 200,
				"parent": mapstr.M{
					"pid": 100,
				},
			},
		},
	}
	prereq := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         100,
				Tgid:        100,
				Vpid:        0,
				Ppid:        0,
				Pgid:        0,
				Sid:         1,
			},
		},
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         200,
				Tgid:        200,
				Vpid:        0,
				Ppid:        100,
				Pgid:        0,
				Sid:         100,
			},
		},
	}
	expected := procfs.ProcessInfo{
		PIDs: types.PIDInfo{
			Tid:  200,
			Tgid: 200,
			Ppid: 100,
			Pgid: 0,
			Sid:  100,
		},
	}

	reader := procfs.NewMockReader()
	db, err := processdb.NewDB(reader, logger)
	require.Nil(t, err)
	for _, entry := range prereq {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}
	db.ScrapeProcfs()

	provider, err := NewProvider(context.TODO(), &logger, db, reader, "process.pid")
	require.Nil(t, err, "error creating provider")

	err = provider.SyncDB(&event, expected.PIDs.Tgid)
	require.Nil(t, err)

	actual, err := db.GetProcess(pid)
	if err != nil {
		require.Fail(t, "pid not found in db")
	}

	require.Equal(t, expected.PIDs.Sid, actual.SessionLeader.PID)
}

func TestSetSidSessionLeaderNotScraped(t *testing.T) {
	var pid uint32 = 200
	event := beat.Event{
		Timestamp: timestamp,
		Fields: mapstr.M{
			"auditd": mapstr.M{
				"data": mapstr.M{
					"a0":      "1",
					"a1":      "ffffeb535e38",
					"a2":      "ffffeb535e48",
					"a3":      "410134",
					"arch":    "aarch64",
					"exit":    "200",
					"syscall": "setsid",
					"tty":     "pts4",
				},
				"result": "success",
			},
			"process": mapstr.M{
				"pid": 200,
				"parent": mapstr.M{
					"pid": 100,
				},
			},
		},
	}
	prereq := []procfs.ProcessInfo{
		{
			PIDs: types.PIDInfo{
				StartTimeNS: 0,
				Tid:         200,
				Tgid:        200,
				Vpid:        0,
				Ppid:        100,
				Pgid:        0,
				Sid:         100,
			},
		},
	}
	expected := procfs.ProcessInfo{
		PIDs: types.PIDInfo{
			Tid:  200,
			Tgid: 200,
			Ppid: 100,
			Pgid: 0,
			Sid:  200,
		},
	}

	reader := procfs.NewMockReader()
	db, err := processdb.NewDB(reader, logger)
	require.Nil(t, err)
	for _, entry := range prereq {
		reader.AddEntry(entry.PIDs.Tgid, entry)
	}
	db.ScrapeProcfs()

	provider, err := NewProvider(context.TODO(), &logger, db, reader, "process.pid")
	require.Nil(t, err, "error creating provider")

	err = provider.SyncDB(&event, expected.PIDs.Tgid)
	require.Nil(t, err)

	actual, err := db.GetProcess(pid)
	if err != nil {
		require.Fail(t, "pid not found in db")
	}

	require.Equal(t, expected.PIDs.Sid, actual.SessionLeader.PID)
}
