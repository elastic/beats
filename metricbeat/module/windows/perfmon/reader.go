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
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	processRegexp = regexp.MustCompile(`(.+?)#[1-9]+`)
)

// Reader will contain the config options
type Reader struct {
	query             Query             // PDH Query
	instanceLabel     map[string]string // Mapping of counter path to key used for the label (e.g. processor.name)
	measurement       map[string]string // Mapping of counter path to key used for the value (e.g. processor.cpu_time).
	executed          bool              // Indicates if the query has been executed.
	log               *logp.Logger      //
	groupMeasurements bool              // Indicates if measurements with the same instance label should be sent in the same event
}

// NewReader creates a new instance of Reader.
func NewReader(config Config) (*Reader, error) {
	var query Query
	if err := query.Open(); err != nil {
		return nil, err
	}

	r := &Reader{
		query:             query,
		instanceLabel:     map[string]string{},
		measurement:       map[string]string{},
		log:               logp.NewLogger("perfmon"),
		groupMeasurements: config.GroupMeasurements,
	}
	for _, counter := range config.CounterConfig {
		childQueries, err := query.ExpandWildCardPath(counter.Query)
		if err != nil {
			if config.IgnoreNECounters {
				switch err {
				case PDH_CSTATUS_NO_COUNTER, PDH_CSTATUS_NO_COUNTERNAME,
					PDH_CSTATUS_NO_INSTANCE, PDH_CSTATUS_NO_OBJECT:
					r.log.Infow("Ignoring non existent counter", "error", err,
						logp.Namespace("perfmon"), "query", counter.Query)
					continue
				}
			} else {
				query.Close()
				return nil, errors.Wrapf(err, `failed to expand counter (query="%v")`, counter.Query)
			}
		}
		// check if the pdhexpandcounterpath/pdhexpandwildcardpath functions have expanded the counter successfully.
		if len(childQueries) == 0 || (len(childQueries) == 1 && strings.Contains(childQueries[0], "*")) {
			query.Close()
			return nil, errors.Errorf(`failed to expand counter (query="%v")`, counter.Query)
		}
		for _, v := range childQueries {
			if err := query.AddCounter(v, counter, len(childQueries) > 1); err != nil {
				query.Close()
				return nil, errors.Wrapf(err, `failed to add counter (query="%v")`, counter.Query)
			}

			r.instanceLabel[v] = counter.InstanceLabel
			r.measurement[v] = counter.MeasurementLabel
		}
	}

	return r, nil
}

// Read executes a query and returns those values in an event.
func (r *Reader) Read() ([]mb.Event, error) {
	if err := r.query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := r.query.GetFormattedCounterValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}

	eventMap := make(map[string]*mb.Event)

	for counterPath, values := range values {
		for ind, val := range values {
			// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
			// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
			if val.Err != nil && !r.executed {
				r.log.Debugw("Ignoring the first measurement because the data isn't ready",
					"error", val.Err, logp.Namespace("perfmon"), "query", counterPath)
				continue
			}

			var eventKey string
			if r.groupMeasurements && val.Err == nil {
				// Send measurements with the same instance label as part of the same event
				eventKey = val.Instance
			} else {
				// Send every measurement as an individual event
				// If a counter contains an error, it will always be sent as an individual event
				eventKey = counterPath + strconv.Itoa(ind)
			}

			// Create a new event if the key doesn't exist in the map
			if _, ok := eventMap[eventKey]; !ok {
				eventMap[eventKey] = &mb.Event{
					MetricSetFields: common.MapStr{},
					Error:           errors.Wrapf(val.Err, "failed on query=%v", counterPath),
				}
				if val.Instance != "" {
					//will ignore instance counter
					if ok, match := matchesParentProcess(val.Instance); ok {
						eventMap[eventKey].MetricSetFields.Put(r.instanceLabel[counterPath], match)
					} else {
						eventMap[eventKey].MetricSetFields.Put(r.instanceLabel[counterPath], val.Instance)
					}
				}
			}
			event := eventMap[eventKey]
			if val.Measurement != nil {
				event.MetricSetFields.Put(r.measurement[counterPath], val.Measurement)
			} else {
				event.MetricSetFields.Put(r.measurement[counterPath], 0)
			}
		}
	}

	// Write the values into the map.
	events := make([]mb.Event, 0, len(eventMap))
	for _, val := range eventMap {
		events = append(events, *val)
	}

	r.executed = true
	return events, nil
}

// Close will close the PDH query for now.
func (r *Reader) Close() error {
	return r.query.Close()
}

// matchParentProcess will try to get the parent process name
func matchesParentProcess(instanceName string) (bool, string) {
	matches := processRegexp.FindStringSubmatch(instanceName)
	if len(matches) == 2 {
		return true, matches[1]
	}
	return false, instanceName
}
