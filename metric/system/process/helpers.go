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

package process

import (
	"math"
	"time"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/numcpu"
)

// unixTimeMsToTime converts a unix time given in milliseconds since Unix epoch
// to a typeconv.Time value.
func unixTimeMsToTime(unixTimeMs uint64) string {
	return typeconv.Time(time.Unix(0, int64(unixTimeMs*1000000))).String()
}

func stripNullByte(buf []byte) string { //nolint: deadcode,unused,nolintlint // it is used in platform specific code
	return string(buf[0 : len(buf)-1])
}

func stripNullByteRaw(buf []byte) []byte { //nolint: deadcode,unused,nolintlint // it is used in platform specific code
	return buf[0 : len(buf)-1]
}

// GetProcMemPercentage returns process memory usage as a percent of total memory usage
func GetProcMemPercentage(proc ProcState, totalPhyMem uint64) opt.Float {
	if totalPhyMem == 0 {
		return opt.NewFloatNone()
	}

	perc := (float64(proc.Memory.Rss.Bytes.ValueOr(0)) / float64(totalPhyMem))

	return opt.FloatWith(metric.Round(perc))
}

// isProcessInSlice looks up proc in the processes slice and returns if
// found or not
func isProcessInSlice(processes []ProcState, proc *ProcState) bool {
	for _, p := range processes {
		if p.Pid == proc.Pid {
			return true
		}
	}
	return false
}

// GetProcCPUPercentage returns the percentage of total CPU time consumed by
// the process during the period between the given samples. Two percentages are
// returned (these must be multiplied by 100). The first is a normalized based
// on the number of cores such that the value ranges on [0, 1]. The second is
// not normalized and the value ranges on [0, number_of_cores].
//
// Implementation note: The total system CPU time (including idle) is not
// provided so this method will resort to using the difference in wall-clock
// time multiplied by the number of cores as the total amount of CPU time
// available between samples. This could result in incorrect percentages if the
// wall-clock is adjusted (prior to Go 1.9) or the machine is suspended.
func GetProcCPUPercentage(s0, s1 ProcState) ProcState {
	// Skip if we're missing the total ticks
	if s0.CPU.Total.Ticks.IsZero() || s1.CPU.Total.Ticks.IsZero() {
		return s1
	}

	timeDelta := s1.SampleTime.Sub(s0.SampleTime)
	timeDeltaDur := timeDelta / time.Millisecond
	totalCPUDeltaMillis := int64(s1.CPU.Total.Ticks.ValueOr(0) - s0.CPU.Total.Ticks.ValueOr(0))

	pct := float64(totalCPUDeltaMillis) / float64(timeDeltaDur)
	// In theory this can only happen if the time delta is 0, which is unlikely but possible.
	// With all the type conversion and non-integer math, this is probably the safest way to check.
	if math.IsNaN(pct) {
		return s1
	}
	normalizedPct := pct / float64(numcpu.NumCPU())

	s1.CPU.Total.Norm.Pct = opt.FloatWith(metric.Round(normalizedPct))
	s1.CPU.Total.Pct = opt.FloatWith(metric.Round(pct))

	return s1

}

// NonFatalErr indicates an error occurred during metrics
// collection, however the metrics already
// gathered and returned are still valid.
// This error can be safely ignored, this will result
// in having partial metrics for a process rather than
// no metrics at all.
//
// It was introduced to allow for partial metrics collection
// on privileged process on Windows.
type NonFatalErr struct {
	Err error
}

func (c NonFatalErr) Error() string {
	if c.Err != nil {
		return "non fatal error; reporting partial metrics: " + c.Err.Error()
	}
	return "non fatal error"
}

func (c NonFatalErr) Is(other error) bool {
	_, is := other.(NonFatalErr)
	return is
}

func (c NonFatalErr) Unwrap() error {
	return c.Err
}

// Wraps a NonFatalError around a generic error, if given error is non-fatal in nature
func toNonFatal(err error) error {
	if err == nil {
		return nil
	}
	if !isNonFatal(err) {
		return err
	}
	return NonFatalErr{Err: err}
}
