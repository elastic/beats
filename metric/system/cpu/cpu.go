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

package cpu

import (
	"context"

	"github.com/shirou/gopsutil/v4/load"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/numcpu"
)

// Load returns CPU load information for the previous 1, 5, and 15 minute
// periods.
// Deprecated: use LoadWithLogger
func Load() (*LoadMetrics, error) {
	return LoadWithLogger(logp.NewLogger(""))
}

// LoadWithLogger returns CPU load information for the previous 1, 5, and 15 minute
// periods.
func LoadWithLogger(logger *logp.Logger) (*LoadMetrics, error) {
	avg, err := load.Avg()
	if err != nil {
		return nil, err
	}

	return &LoadMetrics{avg, logger}, nil
}

func LoadWithContextAndLogger(ctx context.Context, logger *logp.Logger) (*LoadMetrics, error) {
	avg, err := load.AvgWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return &LoadMetrics{avg, logger}, nil
}

// LoadMetrics stores the sampled load average values of the host.
type LoadMetrics struct {
	sample *load.AvgStat
	logger *logp.Logger
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
		OneMinute:     metric.Round(m.sample.Load1),
		FiveMinute:    metric.Round(m.sample.Load5),
		FifteenMinute: metric.Round(m.sample.Load15),
	}
}

// NormalizedAverages return the CPU load averages normalized by the NumCPU.
// These values should range from 0 to 1.
func (m *LoadMetrics) NormalizedAverages() LoadAverages {
	cpus := numcpu.NumCPUWithLogger(m.logger)
	return LoadAverages{
		OneMinute:     metric.Round(m.sample.Load1 / float64(cpus)),
		FiveMinute:    metric.Round(m.sample.Load5 / float64(cpus)),
		FifteenMinute: metric.Round(m.sample.Load15 / float64(cpus)),
	}
}
