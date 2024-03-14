// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package timeutils

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/procfs"
	"github.com/tklauser/go-sysconf"
)

var (
	bootTime       time.Time
	ticksPerSecond uint64
	initError      error
	once           sync.Once
)

func initialize() {
	var err error
	bootTime, err = getBootTime()
	if err != nil {
		initError = err
		return
	}

	ticksPerSecond, err = getTicksPerSecond()
	if err != nil {
		initError = err
	}
}

func getBootTime() (time.Time, error) {
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get procfs: %w", err)
	}

	stat, err := fs.Stat()
	if err != nil {
		return time.Time{}, fmt.Errorf("could not read /proc/stat: %w", err)
	}
	return time.Unix(int64(stat.BootTime), 0), nil
}

func getTicksPerSecond() (uint64, error) {
	tps, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err != nil {
		return 0, fmt.Errorf("sysconf(SC_CLK_TCK) failed: %w", err)
	}
	return uint64(tps), nil
}

func TicksToNs(ticks uint64) uint64 {
	once.Do(initialize)
	if initError != nil {
		return 0
	}
	return ticks * uint64(time.Second.Nanoseconds()) / ticksPerSecond
}

func TimeFromNsSinceBoot(t time.Duration) *time.Time {
	once.Do(initialize)
	if initError != nil {
		return &time.Time{}
	}
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
	once.Do(initialize)
	if initError != nil {
		return 0
	}
	return time.Duration(timeNs).Truncate(time.Second / time.Duration(ticksPerSecond))
}
