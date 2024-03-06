// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package sys

import (
	"sync"
	"time"

	"github.com/prometheus/procfs"
	"github.com/tklauser/go-sysconf"
)

var (
	bootTimeOnce struct {
		sync.Once
		value time.Time
		err   error
	}
	ticksPerSecondOnce struct {
		sync.Once
		value uint64
		err   error
	}
)

func BootTime() (time.Time, error) {
	bootTimeOnce.Do(func() {
		var (
			fs   procfs.FS
			stat procfs.Stat
		)
		fs, bootTimeOnce.err = procfs.NewDefaultFS()
		if bootTimeOnce.err != nil {
			return
		}

		stat, bootTimeOnce.err = fs.Stat()
		if bootTimeOnce.err != nil {
			return
		}
		bootTimeOnce.value = time.Unix(int64(stat.BootTime), 0)
	})
	return bootTimeOnce.value, bootTimeOnce.err
}

func TicksPerSecond() (uint64, error) {
	ticksPerSecondOnce.Do(func() {
		var tps int64
		tps, ticksPerSecondOnce.err = sysconf.Sysconf(sysconf.SC_CLK_TCK)
		if ticksPerSecondOnce.err != nil {
			return
		}
		ticksPerSecondOnce.value = uint64(tps)
	})
	return ticksPerSecondOnce.value, ticksPerSecondOnce.err
}

func TicksToNs(ticks uint64) (uint64, error) {
	tps, err := TicksPerSecond()
	if err != nil {
		return 0, err
	}

	return ticks * uint64(time.Second.Nanoseconds()) / tps, nil
}

func TimeFromNsSinceBoot(ns uint64) (time.Time, error) {
	bt, err := BootTime()
	if err != nil {
		return time.Time{}, err
	}

	reduced, err := reduceTimestampPrecision(ns)
	if err != nil {
		return time.Time{}, err
	}

	return bt.Add(time.Duration(reduced)), nil
}

// When generating an `entity_id` in ECS we need to reduce the precision of a
// process's start time to that of procfs. Process start times can come from either
// eBPF (high precision) or other sources. We must reduce them all to the
// lowest common denominator such that entity ID's generated are always consistent.
func reduceTimestampPrecision(ns uint64) (uint64, error) {
	tps, err := TicksPerSecond()
	if err != nil {
		return 0, err
	}

	return ns - (ns % (uint64(time.Second.Nanoseconds()) / tps)), nil
}
