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

//go:build darwin || freebsd || linux || windows || aix
// +build darwin freebsd linux windows aix

package process_summary

import (
	"runtime"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	sigar "github.com/elastic/gosigar"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("system", "process_summary", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	pids := sigar.ProcList{}
	err := pids.Get()
	if err != nil {
		return errors.Wrap(err, "failed to fetch the list of PIDs")
	}

	var summary struct {
		sleeping int
		running  int
		idle     int
		stopped  int
		zombie   int
		unknown  int
		dead     int
	}

	for _, pid := range pids.List {
		state := sigar.ProcState{}
		err = state.Get(pid)
		if err != nil {
			summary.unknown++
			continue
		}

		switch byte(state.State) {
		case 'S':
			summary.sleeping++
		case 'R':
			summary.running++
		case 'D':
			summary.idle++
		case 'I':
			summary.idle++
		case 'T':
			summary.stopped++
		case 'Z':
			summary.zombie++
		case 'X':
			summary.dead++
		default:
			logp.Err("Unknown or unexpected state <%c> for process with pid %d", state.State, pid)
			summary.unknown++
		}
	}

	event := common.MapStr{}
	if runtime.GOOS == "windows" {
		event = common.MapStr{
			"total":    len(pids.List),
			"sleeping": summary.sleeping,
			"running":  summary.running,
			"unknown":  summary.unknown,
		}
	} else {
		event = common.MapStr{
			"total":    len(pids.List),
			"sleeping": summary.sleeping,
			"running":  summary.running,
			"idle":     summary.idle,
			"stopped":  summary.stopped,
			"zombie":   summary.zombie,
			"unknown":  summary.unknown,
			"dead":     summary.dead,
		}
	}

	r.Event(mb.Event{
		// change the name space to use . instead of _
		Namespace:       "system.process.summary",
		MetricSetFields: event,
	})

	return nil
}
