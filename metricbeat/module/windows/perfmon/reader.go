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

	"github.com/elastic/beats/metricbeat/helper/windows/pdh"

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
	query         pdh.Query         // PDH Query
	instanceLabel map[string]string // Mapping of counter path to key used for the label (e.g. processor.name)
	measurement   map[string]string // Mapping of counter path to key used for the value (e.g. processor.cpu_time).
	executed      bool              // Indicates if the query has been executed.
	log           *logp.Logger      //
	config        Config            // Metricset configuration
}

// NewReader creates a new instance of Reader.
func NewReader(config Config) (*Reader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	r := &Reader{
		query:         query,
		instanceLabel: map[string]string{},
		measurement:   map[string]string{},
		log:           logp.NewLogger("perfmon"),
		config:        config,
	}
	for _, counter := range config.CounterConfig {
		childQueries, err := query.GetCounterPaths(counter.Query)
		if err != nil {
			if config.IgnoreNECounters {
				switch err {
				case pdh.PDH_CSTATUS_NO_COUNTER, pdh.PDH_CSTATUS_NO_COUNTERNAME,
					pdh.PDH_CSTATUS_NO_INSTANCE, pdh.PDH_CSTATUS_NO_OBJECT:
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
			// covering cases when PdhExpandWildCardPathW returns no counter paths or is unable to expand and the ignore_non_existent_counters flag is set
			if config.IgnoreNECounters {
				r.log.Infow("Ignoring non existent counter", "initial query", counter.Query,
					logp.Namespace("perfmon"), "expanded query", childQueries)
				continue
			}
			return nil, errors.Errorf(`failed to expand counter (query="%v")`, counter.Query)
		}
		for _, v := range childQueries {
			if err := query.AddCounter(v, counter.InstanceName, counter.Format, len(childQueries) > 1); err != nil {
				return nil, errors.Wrapf(err, `failed to add counter (query="%v")`, counter.Query)
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
	for _, counter := range r.config.CounterConfig {
		childQueries, err := r.query.GetCounterPaths(counter.Query)
		if err != nil {
			if r.config.IgnoreNECounters {
				switch err {
				case pdh.PDH_CSTATUS_NO_COUNTER, pdh.PDH_CSTATUS_NO_COUNTERNAME,
					pdh.PDH_CSTATUS_NO_INSTANCE, pdh.PDH_CSTATUS_NO_OBJECT:
					r.log.Infow("Ignoring non existent counter", "error", err,
						logp.Namespace("perfmon"), "query", counter.Query)
					continue
				}
			} else {
				return errors.Wrapf(err, `failed to expand counter (query="%v")`, counter.Query)
			}
		}
		newCounters = append(newCounters, childQueries...)
		// there are cases when the ExpandWildCardPath will retrieve a successful status but not an expanded query so we need to check for the size of the list
		if err == nil && len(childQueries) >= 1 && !strings.Contains(childQueries[0], "*") {
			for _, v := range childQueries {
				if err := r.query.AddCounter(v, counter.InstanceName, counter.Format, len(childQueries) > 1); err != nil {
					return errors.Wrapf(err, "failed to add counter (query='%v')", counter.Query)
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
func (this *Reader) Read() ([]mb.Event, error) {
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := this.query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := this.query.GetFormattedCounterValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}
	var events []mb.Event
	if this.config.GroupAllCounters {
		if event := this.groupToEvent(values); event.MetricSetFields != nil {
			events = append(events, event)
		}
	}else {
		if grouped := this.groupToEvents(values); len(grouped)>0 {
			events = append(events, grouped...)
		}
	}

	this.executed = true
	return events, nil
}

func (this *Reader) groupToEvents(counters map[string][]pdh.CounterValue) []mb.Event {
	eventMap := make(map[string]*mb.Event)
	for counterPath, values := range counters {
		for ind, val := range values {
			// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
			// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
			if val.Err != nil && !this.executed {
				this.log.Debugw("Ignoring the first measurement because the data isn't ready",
					"error", val.Err, logp.Namespace("perfmon"), "query", counterPath)
				continue
			}
			var eventKey string
			if this.config.GroupMeasurements && val.Err == nil {
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
				if val.Instance != "" && this.instanceLabel[counterPath] != "" {
					//will ignore instance counter
					if ok, match := matchesParentProcess(val.Instance); ok {
						eventMap[eventKey].MetricSetFields.Put(this.instanceLabel[counterPath], match)
					} else {
						eventMap[eventKey].MetricSetFields.Put(this.instanceLabel[counterPath], val.Instance)
					}
				}
			}
			event := eventMap[eventKey]
			if val.Measurement != nil {
				event.MetricSetFields.Put(this.measurement[counterPath], val.Measurement)
			} else {
				event.MetricSetFields.Put(this.measurement[counterPath], 0)
			}
		}
	}

	// Write the values into the map.
	events := make([]mb.Event, 0, len(eventMap))
	for _, val := range eventMap {
		events = append(events, *val)
	}
	return events
}

func (this *Reader)groupToEvent(counters map[string][]pdh.CounterValue) mb.Event {
	event := mb.Event{
		MetricSetFields: common.MapStr{},
	}
	grouped:= make(map[string][]common.MapStr)
	for counterPath, values := range counters {
		for ind, val := range values {
			// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
			// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
			if val.Err != nil && !this.executed {
				this.log.Debugw("Ignoring the first measurement because the data isn't ready",
					"error", val.Err, logp.Namespace("perfmon"), "query", counterPath)
				continue
			}
			var eventKey string
			if this.config.GroupMeasurements && val.Err == nil {
				// Send measurements with the same instance label as part of the same event
				eventKey = val.Instance
			} else {
				// Send every measurement as an individual event
				// If a counter contains an error, it will always be sent as an individual event
				eventKey = counterPath + strconv.Itoa(ind)
			}
			// Create a new event if the key doesn't exist in the map
			if _, ok := grouped[eventKey]; !ok {
				grouped[eventKey] = make([]common.MapStr,0)
					//[]common.MapStr {{"Error":errors.Wrapf(val.Err, "failed on query=%v", counterPath)}}
			}

			if val.Measurement != nil {
				grouped[eventKey] = append(grouped[eventKey], common.MapStr{this.measurement[counterPath]: val.Measurement})
			} else {
				grouped[eventKey] = append(grouped[eventKey], common.MapStr{this.measurement[counterPath]: 0})
			}
		}
	}

	return event
}

// Close will close the PDH query for now.
func (this *Reader) Close() error {
	return this.query.Close()
}

// matchParentProcess will try to get the parent process name
func matchesParentProcess(instanceName string) (bool, string) {
	matches := processRegexp.FindStringSubmatch(instanceName)
	if len(matches) == 2 {
		return true, matches[1]
	}
	return false, instanceName
}
