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

// +build windows

package perfmon

import (
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/pkg/errors"
	"strings"
)

const replaceUpperCaseRegex = `(?:[^A-Z_\W])([A-Z])[^A-Z]`

// Config for the windows perfmon metricset.
type Config struct {
	IgnoreNECounters   bool      `config:"perfmon.ignore_non_existent_counters"`
	GroupMeasurements  bool      `config:"perfmon.group_measurements_by_instance"`
	Counters           []Counter `config:"perfmon.counters"`
	Queries            []Query   `config:"perfmon.queries"`
	GroupAllCountersTo string    `config:"perfmon.group_all_counter"`
	MetricFormat       bool
}

// Counter for the perfmon counters (old implementation deprecated).
type Counter struct {
	InstanceLabel    string `config:"instance_label"`
	InstanceName     string `config:"instance_name"`
	MeasurementLabel string `config:"measurement_label" validate:"required"`
	Query            string `config:"query"             validate:"required"`
	Format           string `config:"format"`
}

// QueryConfig for perfmon queries. This will be used as the new configuration format
type Query struct {
	Name     string         `config:"object" validate:"required"`
	Field    string         `config:"field"`
	Instance []string       `config:"instance"`
	Counters []QueryCounter `config:"counters" validate:"required"`
}

// QueryConfigCounter for perfmon queries. This will be used as the new configuration format
type QueryCounter struct {
	Name   string `config:"name" validate:"required"`
	Field  string `config:"field"`
	Format string `config:"format"`
}

func (conf *Config) ValidateConfig() error {
	if len(conf.Counters) == 0 && len(conf.Queries) == 0 {
		return errors.New("no perfmon counters or queries have been configured")
	}
	if len(conf.Counters) > 0 {
		cfgwarn.Deprecate("8.0", "perfmon.counters configuration option is deprecated and will be remove in the future major version, we advise using the perfmon.queries configuration option instead")
	}

	if len(conf.Queries) > 0 {
		conf.MetricFormat = true
	}

	// add default format in the config
	for i, value := range conf.Counters {
		form := strings.ToLower(value.Format)
		switch form {
		case "", "float":
			conf.Counters[i].Format = "float"
		case "long", "large":
		default:
			return errors.Errorf("initialization failed: format '%s' "+
				"for counter '%s' is invalid (must be float, large or long)",
				value.Format, value.InstanceLabel)
		}
	}
	for _, value := range conf.Queries {
		for i, q := range value.Counters {
			form := strings.ToLower(q.Format)
			switch form {
			case "", "float":
				value.Counters[i].Format = "float"
			case "long", "large":
			default:
				return errors.Errorf("initialization failed: format '%s' "+
					"for counter '%s' is invalid (must be float, large or long)",
					q.Format, q.Field)
			}
		}

	}
	return nil
}
