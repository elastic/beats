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

package entropy

import (
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "entropy", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	randomPath string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system entropy metricset is beta.")

	totalPath := paths.Resolve(paths.Hostfs, "/proc/sys/kernel/random")

	return &MetricSet{
		BaseMetricSet: base,
		randomPath:    totalPath,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	entropy, err := getEntropyData(path.Join(m.randomPath, "entropy_avail"))
	if err != nil {
		return errors.Wrap(err, "error getting entropy")
	}
	poolsize, err := getEntropyData(path.Join(m.randomPath, "poolsize"))
	if err != nil {
		return errors.Wrap(err, "error getting poolsize")
	}
	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"available_bits": entropy,
			"pct":            float64(entropy) / float64(poolsize),
		},
	})

	return nil
}

func getEntropyData(path string) (int, error) {
	//This will be a number in the range 0 to 4096.
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, errors.Wrap(err, "error reading from random")
	}

	intval, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "error parsing from random")
	}

	return int(intval), nil
}
