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

package core

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	metrics "github.com/elastic/beats/v8/metricbeat/internal/metrics/cpu"
)

// Core metric types.
const (
	percentages = "percentages"
	ticks       = "ticks"
)

// Config for the system core metricset.
type Config struct {
	Metrics  []string `config:"core.metrics"`
	CPUTicks *bool    `config:"cpu_ticks"` // Deprecated.
}

// Validate validates the core config.
func (c Config) Validate() (metrics.MetricOpts, error) {
	opts := metrics.MetricOpts{}
	if c.CPUTicks != nil {
		cfgwarn.Deprecate("6.1.0", "cpu_ticks is deprecated. Add 'ticks' to the core.metrics list.")
	}

	if len(c.Metrics) == 0 {
		return opts, errors.New("core.metrics cannot be empty")
	}

	for _, metric := range c.Metrics {
		switch strings.ToLower(metric) {
		case percentages:
			opts.Percentages = true
		case ticks:
			opts.Ticks = true
		default:
			return opts, errors.Errorf("invalid core.metrics value '%v' (valid "+
				"options are %v and %v)", metric, percentages, ticks)
		}
	}

	return opts, nil
}

var defaultConfig = Config{
	Metrics: []string{percentages},
}
