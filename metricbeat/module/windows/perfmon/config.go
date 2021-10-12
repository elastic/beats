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

//go:build windows
// +build windows

package perfmon

import (
	"time"

	"github.com/pkg/errors"
)

var allowedFormats = []string{"float", "large", "long"}

// Config for the windows perfmon metricset.
type Config struct {
	Period                  time.Duration `config:"period" validate:"required"`
	IgnoreNECounters        bool          `config:"perfmon.ignore_non_existent_counters"`
	GroupMeasurements       bool          `config:"perfmon.group_measurements_by_instance"`
	RefreshWildcardCounters bool          `config:"perfmon.refresh_wildcard_counters"`
	Queries                 []Query       `config:"perfmon.queries"`
	GroupAllCountersTo      string        `config:"perfmon.group_all_counter"`
}

// QueryConfig for perfmon queries. This will be used as the new configuration format
type Query struct {
	Name      string         `config:"object" validate:"required"`
	Field     string         `config:"field"`
	Instance  []string       `config:"instance"`
	Counters  []QueryCounter `config:"counters" validate:"required,nonzero"`
	Namespace string         `config:"namespace"`
}

// QueryConfigCounter for perfmon queries. This will be used as the new configuration format
type QueryCounter struct {
	Name   string `config:"name" validate:"required"`
	Field  string `config:"field"`
	Format string `config:"format"`
}

func (query *Query) InitDefaults() {
	query.Namespace = "metrics"
}

func (counter *QueryCounter) InitDefaults() {
	counter.Format = "float"
}

func (counter *QueryCounter) Validate() error {
	if !isValidFormat(counter.Format) {
		return errors.Errorf("initialization failed: format '%s' "+
			"for counter '%s' is invalid (must be float, large or long)",
			counter.Format, counter.Name)
	}
	return nil
}

func (conf *Config) Validate() error {
	if len(conf.Queries) == 0 {
		return errors.New("No perfmon queries have been configured. Please follow documentation on allowed configuration settings (perfmon.counters configuration option has been deprecated and is removed in 8.0, perfmon.queries configuration option can be used instead). ")
	}
	return nil
}

func isValidFormat(format string) bool {
	for _, form := range allowedFormats {
		if form == format {
			return true
		}
	}
	return false
}
