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

//go:build darwin || freebsd || linux || openbsd || windows || aix
// +build darwin freebsd linux openbsd windows aix

package cpu

import (
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/metric/system/numcpu"
	sigar "github.com/menderesk/gosigar"
)

// Load returns CPU load information for the previous 1, 5, and 15 minute
// periods.
func Load() (*LoadMetrics, error) {
	load := &sigar.LoadAverage{}
	if err := load.Get(); err != nil {
		return nil, err
	}

	return &LoadMetrics{load}, nil
}

// LoadMetrics stores the sampled load average values of the host.
type LoadMetrics struct {
	sample *sigar.LoadAverage
}

// LoadAverages stores the values of load averages of the last 1, 5 and 15 minutes.
type LoadAverages struct {
	OneMinute     float64
	FiveMinute    float64
	FifteenMinute float64
}

// Averages return the CPU load averages. These values should range from
// 0 to NumCPU.
func (m *LoadMetrics) Averages() LoadAverages {
	return LoadAverages{
		OneMinute:     common.Round(m.sample.One, common.DefaultDecimalPlacesCount),
		FiveMinute:    common.Round(m.sample.Five, common.DefaultDecimalPlacesCount),
		FifteenMinute: common.Round(m.sample.Fifteen, common.DefaultDecimalPlacesCount),
	}
}

// NormalizedAverages return the CPU load averages normalized by the NumCPU.
// These values should range from 0 to 1.
func (m *LoadMetrics) NormalizedAverages() LoadAverages {
	cpus := numcpu.NumCPU()
	return LoadAverages{
		OneMinute:     common.Round(m.sample.One/float64(cpus), common.DefaultDecimalPlacesCount),
		FiveMinute:    common.Round(m.sample.Five/float64(cpus), common.DefaultDecimalPlacesCount),
		FifteenMinute: common.Round(m.sample.Fifteen/float64(cpus), common.DefaultDecimalPlacesCount),
	}
}
