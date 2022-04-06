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
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/numcpu"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// unixTimeMsToTime converts a unix time given in milliseconds since Unix epoch
// to a common.Time value.
func unixTimeMsToTime(unixTimeMs uint64) string {
	return common.Time(time.Unix(0, int64(unixTimeMs*1000000))).String()
}

func stripNullByte(buf []byte) string {
	return string(buf[0 : len(buf)-1])
}

func stripNullByteRaw(buf []byte) []byte {
	return buf[0 : len(buf)-1]
}

// GetProcMemPercentage returns process memory usage as a percent of total memory usage
func GetProcMemPercentage(proc ProcState, totalPhyMem uint64) opt.Float {
	if totalPhyMem == 0 {
		return opt.NewFloatNone()
	}

	perc := (float64(proc.Memory.Rss.Bytes.ValueOr(0)) / float64(totalPhyMem))

	return opt.FloatWith(common.Round(perc, 4))
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
	timeDeltaMillis := timeDelta / time.Millisecond
	totalCPUDeltaMillis := int64(s1.CPU.Total.Ticks.ValueOr(0) - s0.CPU.Total.Ticks.ValueOr(0))

	pct := float64(totalCPUDeltaMillis) / float64(timeDeltaMillis)
	normalizedPct := pct / float64(numcpu.NumCPU())

	s1.CPU.Total.Norm.Pct = opt.FloatWith(common.Round(normalizedPct, common.DefaultDecimalPlacesCount))
	s1.CPU.Total.Pct = opt.FloatWith(common.Round(pct, common.DefaultDecimalPlacesCount))
	s1.CPU.Total.Value = opt.FloatWith(common.Round(float64(s1.CPU.Total.Ticks.ValueOr(0)), common.DefaultDecimalPlacesCount))

	return s1

}
