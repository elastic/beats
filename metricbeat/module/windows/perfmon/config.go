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
	"github.com/pkg/errors"
	"strings"
)

// Config for the windows perfmon metricset.
type Config struct {
	IgnoreNECounters   bool            `config:"perfmon.ignore_non_existent_counters"`
	GroupMeasurements  bool            `config:"perfmon.group_measurements_by_instance"`
	CounterConfig      []CounterConfig `config:"perfmon.counters" validate:"required"`
	QueryConfig        []QueryConfig   `config:"perfmon.queries" validate:"required"`
	GroupAllCountersTo string          `config:"perfmon.group_all_counter"`
	MetricFormat       bool
}

// CounterConfig for perfmon counters.
type CounterConfig struct {
	InstanceLabel    string `config:"instance_label"`
	InstanceName     string `config:"instance_name"`
	MeasurementLabel string `config:"measurement_label" validate:"required"`
	Query            string `config:"query"             validate:"required"`
	Format           string `config:"format"`
}

// QueryConfig for perfmon queries. This will be used as the new configuration format
type QueryConfig struct {
	Name     string               `config:"object" validate:"required"`
	Field    string               `config:"field"`
	Instance string               `config:"instance"`
	Counters []QueryConfigCounter `config:"counters" validate:"required"`
}

// QueryConfigCounter for perfmon queries. This will be used as the new configuration format
type QueryConfigCounter struct {
	Name   string `config:"counters" validate:"required"`
	Field  string `config:"field"`
	Format string `config:"format"`
}

func (conf *Config) ValidateConfig() error {
	if len(conf.CounterConfig) == 0 && len(conf.QueryConfig) == 0 {
		return errors.New("no perfmon counters or queries have been configured")
	}
	if len(conf.QueryConfig) > 0 {
		conf.MetricFormat = true
		conf.CounterConfig = []CounterConfig{}
		for _, query := range conf.QueryConfig {
			for _, counter := range query.Counters {
				var counterConf = CounterConfig{
					InstanceLabel:    "instance",
					InstanceName:     query.Instance,
					MeasurementLabel: counter.Field,
					Query:            query.Name,
					Format:           counter.Format,
				}
				conf.CounterConfig = append(conf.CounterConfig, counterConf)
			}

		}

	}

	// add default format in the config
	for _, value := range conf.CounterConfig {
		form := strings.ToLower(value.Format)
		switch form {
		case "", "float":
			value.Format = "float"
		case "long", "large":
		default:
			return errors.Errorf("initialization failed: format '%s' "+
				"for counter '%s' is invalid (must be float, large or long)",
				value.Format, value.InstanceLabel)
		}
	}
	for _, value := range conf.QueryConfig {
		for _, path := range value.Counters {
			form := strings.ToLower(path.Format)
			switch form {
			case "", "float":
				path.Format = "float"
			case "long", "large":
			default:
				return errors.Errorf("initialization failed: format '%s' "+
					"for counter '%s' is invalid (must be float, large or long)",
					path.Format, value.Name)
			}
		}

	}

	return nil
}
