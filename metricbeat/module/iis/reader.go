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
	"regexp"
	"strings"

	"github.com/elastic/beats/metricbeat/module/windows/perfmon"

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
	query         perfmon.Query     // PDH Query
	instanceLabel map[string]string // Mapping of counter path to key used for the label (e.g. processor.name)
	measurement   map[string]string // Mapping of counter path to key used for the value (e.g. processor.cpu_time).
	Executed      bool              // Indicates if the query has been executed.
	log           *logp.Logger      //
	counters      []PerformanceCounter
}

// NewReader creates a new instance of Reader.
func NewReader(counters []PerformanceCounter) (*Reader, error) {
	var query perfmon.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	r := &Reader{
		query:         query,
		instanceLabel: map[string]string{},
		measurement:   map[string]string{},
		log:           logp.NewLogger("iis"),
		counters:      counters,
	}
	for _, counter := range counters {
		childQueries, err := query.GetCounterPaths(counter.Path)
		if err != nil {
			if err == perfmon.PDH_CSTATUS_NO_COUNTER || err == perfmon.PDH_CSTATUS_NO_COUNTERNAME || err == perfmon.PDH_CSTATUS_NO_INSTANCE || err == perfmon.PDH_CSTATUS_NO_OBJECT {
				r.log.Infow("Ignoring non existent counter", "error", err,
					logp.Namespace("iis"), "query", counter.Path)
				continue
			} else {
				query.Close()
				return nil, errors.Wrapf(err, `failed to expand counter (query="%v")`, counter.Path)
			}
		}
		// check if the pdhexpandcounterpath/pdhexpandwildcardpath functions have expanded the counter successfully.
		if len(childQueries) == 0 || (len(childQueries) == 1 && strings.Contains(childQueries[0], "*")) {
			// covering cases when PdhExpandWildCardPathW returns no counter paths or is unable to expand and the ignore_non_existent_counters flag is set
			r.log.Infow("Ignoring non existent counter", "initial query", counter.Path,
				logp.Namespace("perfmon"), "expanded query", childQueries)
			continue
		}
		for _, v := range childQueries {
			if err := query.AddCounter(v, counter.InstanceLabel, counter.Format, len(childQueries) > 1); err != nil {
				return nil, errors.Wrapf(err, `failed to add counter (query="%v")`, counter.Path)
			}
			r.instanceLabel[v] = counter.InstanceLabel
			r.measurement[v] = counter.MeasurementLabel
		}
	}

	return r, nil
}

// RefreshCounterPaths will recheck for any new instances and add them to the counter list
func (r *Reader) RefreshCounterPaths() error {
	var newCounters []string
	for _, counter := range r.counters {
		childQueries, err := r.query.GetCounterPaths(counter.Path)
		if err != nil {
			if err == perfmon.PDH_CSTATUS_NO_COUNTER || err == perfmon.PDH_CSTATUS_NO_COUNTERNAME || err == perfmon.PDH_CSTATUS_NO_INSTANCE || err == perfmon.PDH_CSTATUS_NO_OBJECT {
				r.log.Infow("Ignoring non existent counter", "error", err,
					logp.Namespace("iis"), "query", counter.Path)
				continue
			} else {
				return errors.Wrapf(err, `failed to expand counter (query="%v")`, counter.Path)
			}
		}
		newCounters = append(newCounters, childQueries...)
		// there are cases when the ExpandWildCardPath will retrieve a successful status but not an expanded query so we need to check for the size of the list
		if err == nil && len(childQueries) >= 1 && !strings.Contains(childQueries[0], "*") {
			for _, v := range childQueries {
				if err := r.query.AddCounter(v, counter.InstanceLabel, counter.Format, len(childQueries) > 1); err != nil {
					return errors.Wrapf(err, "failed to add counter (query='%v')", counter.Path)
				}
				r.instanceLabel[v] = counter.InstanceLabel
				r.measurement[v] = counter.MeasurementLabel
			}
		}
	}
	err := r.query.RemoveUnusedCounters(newCounters)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}

	return nil
}

// Read executes a query and returns those values in an event.
func (r *Reader) Read(metricsetName string) ([]mb.Event, error) {
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := r.query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := r.query.GetFormattedCounterValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}

	eventMap := make(map[string]*mb.Event)

	for path, counterValues := range values {
		for _, counterValue := range counterValues {
			if counterValue.Err != nil && !r.Executed {
				r.log.Debugw("Ignoring the first measurement because the data isn't ready",
					"error", counterValue.Err, logp.Namespace("iis"), "query", path)
				continue
			}
			if counterValue.Err != nil {
				r.log.Errorw("Error while retrieving counter values",
					"error", counterValue.Err, logp.Namespace("perfmon"), "query", path)
				continue
			}
			var eventKey string
			if metricsetName == "webserver" {
				// Send measurements with the same instance label as part of the same event
				eventKey = metricsetName
			} else {
				// Send every measurement as an individual event
				// If a counter contains an error, it will always be sent as an individual event
				eventKey = counterValue.Instance
			}

			// Create a new event if the key doesn't exist in the map
			if _, ok := eventMap[eventKey]; !ok {
				eventMap[eventKey] = &mb.Event{
					MetricSetFields: common.MapStr{},
				}
				if metricsetName != "webserver" {
					//will ignore instance counter
					if ok, match := matchesParentProcess(counterValue.Instance); ok {
						eventMap[eventKey].MetricSetFields.Put(r.instanceLabel[path], match)
					} else {
						eventMap[eventKey].MetricSetFields.Put(r.instanceLabel[path], counterValue.Instance)
					}
				}
			}
			event := eventMap[eventKey]
			if counterValue.Measurement != nil {
				event.MetricSetFields.Put(r.measurement[path], counterValue.Measurement)
			} else {
				event.MetricSetFields.Put(r.measurement[path], 0)
			}
		}
	}

	// Write the values into the map.
	events := make([]mb.Event, 0, len(eventMap))
	for _, val := range eventMap {
		events = append(events, *val)
	}

	r.Executed = true
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
