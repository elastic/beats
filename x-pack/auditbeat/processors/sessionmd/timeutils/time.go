// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package timeutils

import (
	"fmt"
	"time"

	"github.com/prometheus/procfs"
	"github.com/tklauser/go-sysconf"
)

var (
	bootTime       = mustGetBootTime()
	ticksPerSecond = mustGetTicksPerSecond()
)

func mustGetBootTime() time.Time {
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		panic(fmt.Sprintf("could not get procfs: %v", err))
	}

	stat, err := fs.Stat()
	if err != nil {
		panic(fmt.Sprintf("could not read /proc/stat: %v", err))
	}
	return time.Unix(int64(stat.BootTime), 0)
}

func mustGetTicksPerSecond() uint64 {
	tps, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err != nil {
		panic(fmt.Sprintf("sysconf(SC_CLK_TCK) failed: %v", err))
	}
	return uint64(tps)
}

func TicksToNs(ticks uint64) uint64 {
	return ticks * uint64(time.Second.Nanoseconds()) / ticksPerSecond
}

func TimeFromNsSinceBoot(t time.Duration) *time.Time {
	timestamp := bootTime.Add(t)
	return &timestamp
}

// When generating an `entity_id` in ECS we need to reduce the precision of a
// process's start time to that of procfs. Process start times can come from either
// BPF (high precision) or procfs (lower precision). We must reduce them all to the
// lowest common denominator such that entity ID's generated are always consistent.
//
//   - Timestamps we get from the kernel are in nanosecond precision.
//   - Timestamps we get from procfs are typically 1/100th second precision. We
//     get this precision from `sysconf()`
//   - We store timestamps as nanoseconds, but reduce the precision to 1/100th
//     second
func ReduceTimestampPrecision(timeNs uint64) time.Duration {
	return time.Duration(timeNs).Truncate(time.Second / time.Duration(ticksPerSecond))
}
