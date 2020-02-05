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

package iis

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/windows/pdh"
	"github.com/elastic/beats/metricbeat/mb"
)

// Reader will contain the config options
type Reader struct {
	Query     pdh.Query    // PDH Query
	Instances []Instance   // Mapping of counter path to key used for the label (e.g. processor.name)
	log       *logp.Logger // logger
	hasRun    bool         // will check if the reader has run a first time
}

type Instance struct {
	Name      string
	ProcessId int
	counters  map[string]string
}

// NewReader creates a new instance of Reader.
func NewReader() (*Reader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	reader := &Reader{
		Query: query,
		log:   logp.NewLogger("website"),
	}

	return reader, nil
}

func (re *Reader) InitCounters(nameCounters map[string]string, processIdCounters map[string]string) error {
	var newQueries []string
	for i, instance := range re.Instances {
		re.Instances[i].counters = make(map[string]string)
		for key, value := range nameCounters {
			value = strings.Replace(value, "*", instance.Name, 1)
			if err := re.Query.AddCounter(value, "", "float", true); err != nil {
				return errors.Wrapf(err, `failed to add counter (query="%v")`, value)
			}
			newQueries = append(newQueries, value)
			re.Instances[i].counters[value] = key
		}
		for key, value := range processIdCounters {
			value = strings.Replace(value, "*", string(instance.ProcessId), 1)
			if err := re.Query.AddCounter(value, "", "float", true); err != nil {
				return errors.Wrapf(err, `failed to add counter (query="%v")`, value)
			}
			newQueries = append(newQueries, value)
			re.Instances[i].counters[value] = key
		}
	}
	err := re.Query.RemoveUnusedCounters(newQueries)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}
	return nil
}

// Read executes a query and returns those values in an event.
func (re *Reader) Fetch(nameCounters map[string]string, processIdCounters map[string]string) ([]mb.Event, error) {
	// if the ignore_non_existent_counters flag is set and no valid counter paths are found the Read func will still execute, a check is done before
	if len(re.Query.Counters) == 0 {
		return nil, errors.New("no counters to read")
	}

	// refresh performance counter list
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	// A flag is set if the second call has been executed else refresh will fail (reader.executed)
	if re.hasRun {
		err := re.InitCounters(nameCounters, processIdCounters)
		if err != nil {
			return nil, errors.Wrap(err, "failed retrieving counters")
		}
	}

	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := re.Query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := re.Query.GetFormattedCounterValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}
	events := make(map[string]mb.Event)
	for _, host := range re.Instances {
		events[host.Name] = mb.Event{
			MetricSetFields: common.MapStr{
				"name":              host.Name,
				"worker_process_id": host.ProcessId,
			},
		}
		for counterPath, values := range values {
			for _, val := range values {
				// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
				// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
				if val.Err != nil && !re.hasRun {
					re.log.Debugw("Ignoring the first measurement because the data isn't ready",
						"error", val.Err, logp.Namespace("website"), "query", counterPath)
					continue
				}
				if val.Instance == host.Name || val.Instance == string(host.ProcessId) {
					events[host.Name].MetricSetFields.Put(host.counters[counterPath], val.Measurement)
				}
			}
		}
	}

	re.hasRun = true
	results := make([]mb.Event, 0, len(events))
	for _, val := range events {
		results = append(results, val)
	}
	return results, nil
}

// Close will close the PDH query for now.
func (re *Reader) Close() error {
	return re.Query.Close()
}
