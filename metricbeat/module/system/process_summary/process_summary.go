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
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
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
	sys resolve.Resolver
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	sys := base.Module().(resolve.Resolver)
	return &MetricSet{
		BaseMetricSet: base,
		sys:           sys,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {

	procList, err := process.ListStates(m.sys)
	if err != nil {
		return errors.Wrap(err, "error fetching process list")
	}

	procStates := map[string]int{}
	for _, proc := range procList {
		if count, ok := procStates[string(proc.State)]; ok {
			procStates[string(proc.State)] = count + 1
		} else {
			procStates[string(proc.State)] = 1
		}
	}

	outMap := mapstr.M{}
	typeconv.Convert(&outMap, procStates)
	outMap["total"] = len(procList)
	r.Event(mb.Event{
		// change the name space to use . instead of _
		Namespace:       "system.process.summary",
		MetricSetFields: outMap,
	})

	return nil
}

func threadStats(sys resolve.Resolver) (mapstr.M, error) {
	statPath := sys.ResolveHostFS("/proc/stat")
	procData, err := ioutil.ReadFile(statPath)
	if err != nil {
		return nil, fmt.Errorf("error reading procfs file %s: %w", statPath, err)
	}

	threadData := mapstr.M{}
	for _, line := range strings.Split(string(procData), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.Contains(fields[0], "procs_") {
			procsInt, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Error parsing value %s from %s: %w", fields[0], statPath, err)
			}
			threadData[fields[0]] = procsInt
		}
	}
	return threadData, nil
}
