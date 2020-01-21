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

package website

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/windows/perfmon"
	"github.com/pkg/errors"
	"strings"
)

// Reader will contain the config options
type Reader struct {
	query perfmon.Query // PDH Query
	hosts []string      // Mapping of counter path to key used for the label (e.g. processor.name)
	log   *logp.Logger  // logger
	hasRun bool         // will check if the reader has run a first time
}

// NewReader creates a new instance of Reader.
func NewReader(config Config) (*Reader, error) {
	var query perfmon.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	reader := &Reader{
		query: query,
		log:   logp.NewLogger("website"),
	}
	if err := reader.InitCounters(config.Hosts); err != nil {
		return nil, err
	}
	return reader, nil
}

func (this *Reader) InitCounters(hosts []string) error {
	counters, instances, err := this.query.GetCountersAndInstances("Web Service")
	_ = counters
	if err != nil {
		this.query.Close()
		return err
	}
	this.hosts = filterOnInstances(hosts, instances)
	var newQueries []string
	for _, instance := range this.hosts {
		for _, value := range webserverCounters {
			value = strings.Replace(value, "*", instance, 1)
			if err := this.query.AddCounter(value, "", "float", true); err != nil {
				return errors.Wrapf(err, `failed to add counter (query="%v")`, value)
			}
			newQueries = append(newQueries, value)
		}
	}
	err = this.query.RemoveUnusedCounters(newQueries)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}
	return nil
}

// Read executes a query and returns those values in an event.
func (this *Reader) Fetch() error {
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := this.query.CollectData(); err != nil {
		return errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := this.query.GetFormattedCounterValues()
	if err != nil {
		return errors.Wrap(err, "failed formatting counter values")
	}

	for counterPath, values := range values {
		for _, val := range values {
			// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
			// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).

			if val.Err != nil && !this.hasRun {
				this.log.Debugw("Ignoring the first measurement because the data isn't ready",
					"error", val.Err, logp.Namespace("perfmon"), "query", counterPath)
				continue
			}

		}
	}
	this.hasRun = true
	return nil
}

// Close will close the PDH query for now.
func (this *Reader) Close() error {
	return this.query.Close()
}

func filterOnInstances(hosts []string, instances []string) []string {
	var filtered []string
	// remove _Total and empty instances
	for _, instance := range instances {
		if instance == "_Total" || instance == "" {
			continue
		}
		if containsHost(instance, hosts) {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

func containsHost(item string, array []string) bool {
	// if no hosts specified all instances are selected
	if len(array) == 0 {
		return true
	}
	for _, i := range array {
		if i == item {
			return true
		}
	}
	return false
}
