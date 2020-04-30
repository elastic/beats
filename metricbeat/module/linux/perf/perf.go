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

// +build linux

package perf

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/linux"

	"github.com/hodgesds/perf-utils"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("linux", "perf", New)
}

type eventsConfig struct {
	HardwareEvents bool `config:"hardware"`
	SoftwareEvents bool `config:"software"`
}

type sampleConfig struct {
	ProcessGlob string       `config:"process"`
	Events      eventsConfig `config:"events"`
}

// Config holds the metricset config info for perf
type Config struct {
	SamplePeriod time.Duration  `config:"perf.sample_period" validate:"required,nonzero"`
	Processes    []sampleConfig `config:"perf.processes"`
}

// perfInfo is the "final" sampling data returned from a sample run
// The struct contains metadata from the process along with sampling counters
type perfInfo struct {
	Metadata  common.MapStr
	HwMetrics perf.HardwareProfile
	SwMetrics perf.SoftwareProfile
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	processes  []procInfo
	period     time.Duration
	configData []sampleConfig
	logger     *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The linux perf metricset is beta.")
	logger := logp.NewLogger("perf")
	linuxModule, ok := base.Module().(*linux.Module)
	if !ok {
		return nil, errors.New("unexpected module type")
	}
	config := Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.SamplePeriod > linuxModule.Period {
		return nil, fmt.Errorf("Sample period of %s exceeds metricset period of %s", config.SamplePeriod, linuxModule.Period)
	}

	if config.SamplePeriod == 0 {
		return nil, fmt.Errorf("Sample Period is zero")
	}

	procList, err := matchProcesses(config.Processes)
	if err != nil {
		return nil, errors.Wrap(err, "error gathering processes")
	}

	if len(procList) == 0 {
		logger.Warn("No processes found matching config, will retry")
		procList = make([]procInfo, 0)
	}

	// This perf library is Not So Good and eats errors that come from perf_event_open
	// Do a quck test using a lower-level API to make sure we can actually access perf APIs
	testP, err := perf.NewMinorFaultsProfiler(procList[0].PID, -1)
	defer testP.Close()
	if err != nil {
		return nil, errors.Wrap(err, "error creating profiler for Minor Page faults")
	}

	return &MetricSet{
		BaseMetricSet: base,
		processes:     procList,
		period:        config.SamplePeriod,
		configData:    config.Processes,
		logger:        logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	if len(m.processes) == 0 {
		m.logger.Warn("Retrying search for processes")
		procList, err := matchProcesses(m.configData)
		if err != nil {
			return errors.Wrap(err, "error searching for processes")
		}
		if len(procList) == 0 {
			return fmt.Errorf("No processes match config, retrying")
		}
		m.processes = procList
	}

	perfData, err := runSampleForPeriod(m.processes, m.period)
	if err != nil {
		return errors.Wrap(err, "error running sample")
	}

	for _, process := range perfData {

		event := common.MapStr{}

		if process.SwMetrics.TimeRunning != nil {
			event["software_events"] = process.SwMetrics
		}

		if process.HwMetrics.TimeRunning != nil {
			event["hardware_events"] = process.HwMetrics
		}

		report.Event(mb.Event{
			MetricSetFields: event,
			RootFields: common.MapStr{
				"process": common.MapStr{
					"pid":  process.Metadata["pid"],
					"ppid": process.Metadata["ppid"],
					"pgid": process.Metadata["pgid"],
					"name": process.Metadata["name"],
				},
			},
		})
	}

	return nil
}

// in cases where we're not stopping monitoring, start before we call `runSampleForPeriod`.
// Otherwise call every sample period.
func startMonitor(toSample []procInfo) error {
	for _, pid := range toSample {

		if pid.HardwareProc != nil {
			err := pid.HardwareProc.Start()
			if err != nil {
				return errors.Wrap(err, "error starting HW profiler")
			}
		}

		if pid.SoftwareProc != nil {
			err := pid.SoftwareProc.Start()
			if err != nil {
				return errors.Wrap(err, "error starting SW profiler")
			}
		}

	}
	return nil
}

//TODO: remove the "continuous" thing
// runSampleForPeriod starts a sample, sleeps for a given period, then collects metrics.
func runSampleForPeriod(toSample []procInfo, period time.Duration) ([]perfInfo, error) {

	err := startMonitor(toSample)
	if err != nil {
		return nil, errors.Wrap(err, "error starting monitor")
	}

	time.Sleep(period)

	var metrics = []perfInfo{}

	for _, pid := range toSample {
		// check to make sure the PID is still "live"
		err := pid.checkAndReplace()
		if err != nil {
			return nil, err
		}

		newMetric := perfInfo{}
		newMetric.Metadata = pid.Metadata

		if pid.HardwareProc != nil {
			hwPro, err := pid.HardwareProc.Profile()
			if err != nil {
				return nil, errors.Wrap(err, "error gathering HW profile")
			}
			newMetric.HwMetrics = *hwPro

			pid.HardwareProc.Stop()
			pid.HardwareProc.Reset()
		}

		if pid.SoftwareProc != nil {
			swPro, err := pid.SoftwareProc.Profile()
			if err != nil {
				return nil, errors.Wrap(err, "error gathering SW profile")
			}
			newMetric.SwMetrics = *swPro

			pid.SoftwareProc.Stop()
			pid.SoftwareProc.Reset()

		}

		metrics = append(metrics, newMetric)
	}

	return metrics, nil

}

// Close closes the metricset
func (m *MetricSet) Close() error {
	for _, pid := range m.processes {
		if pid.HardwareProc != nil {
			err := pid.HardwareProc.Stop()
			if err != nil {
				return errors.Wrap(err, "error starting HW profiler")
			}
		}

		if pid.SoftwareProc != nil {
			err := pid.SoftwareProc.Stop()
			if err != nil {
				return errors.Wrap(err, "error starting SW profiler")
			}
		}

	}
	return nil

}
