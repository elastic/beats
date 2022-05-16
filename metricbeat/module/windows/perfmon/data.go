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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var processRegexp = regexp.MustCompile(`(.+?[^\s])(?:#\d+|$)`)

func (re *Reader) groupToEvents(counters map[string][]pdh.CounterValue) []mb.Event {
	eventMap := make(map[string]*mb.Event)
	for counterPath, values := range counters {
		hasCounter, counter := re.getCounter(counterPath)
		if !hasCounter {
			continue
		}

		for ind, val := range values {
			// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
			// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
			if val.Err.Error != nil {
				// The counter has a negative value or the counter was successfully found, but the data returned is not valid.
				// This error can occur if the counter value is less than the previous value. (Because counter values always increment, the counter value rolls over to zero when it reaches its maximum value.)
				// This is not an error that stops the application from running successfully and a positive counter value should be retrieved in the later calls.
				if val.Err.Error == pdh.PDH_CALC_NEGATIVE_VALUE || val.Err.Error == pdh.PDH_INVALID_DATA {
					re.log.Debugw("Counter value retrieval returned",
						"error", val.Err.Error, "cstatus", pdh.PdhErrno(val.Err.CStatus), logp.Namespace("perfmon"), "query", counterPath)
					continue
				}
			}

			var eventKey string
			if re.config.GroupMeasurements && val.Err.Error == nil {
				// Send measurements from the same object with the same instance label as part of the same event
				eventKey = counter.ObjectName + "\\" + val.Instance
			} else {
				// Send every measurement as an individual event
				// If a counter contains an error, it will always be sent as an individual event
				eventKey = counterPath + strconv.Itoa(ind)
			}

			// Create a new event if the key doesn't exist in the map
			if _, ok := eventMap[eventKey]; !ok {
				eventMap[eventKey] = &mb.Event{
					MetricSetFields: mapstr.M{},
					Error:           errors.Wrapf(val.Err.Error, "failed on query=%v", counterPath),
				}
				if val.Instance != "" {
					// will ignore instance index
					if ok, match := matchesParentProcess(val.Instance); ok {
						eventMap[eventKey].MetricSetFields.Put(counter.InstanceField, match)
					} else {
						eventMap[eventKey].MetricSetFields.Put(counter.InstanceField, val.Instance)
					}
				}
			}

			if val.Measurement != nil {
				eventMap[eventKey].MetricSetFields.Put(counter.QueryField, val.Measurement)
			} else {
				eventMap[eventKey].MetricSetFields.Put(counter.QueryField, 0)
			}

			if counter.ObjectField != "" {
				eventMap[eventKey].MetricSetFields.Put(counter.ObjectField, counter.ObjectName)
			}
		}
	}
	// Write the values into the map.
	var events []mb.Event
	for _, val := range eventMap {
		events = append(events, *val)
	}
	return events
}

func (re *Reader) groupToSingleEvent(counters map[string][]pdh.CounterValue) mb.Event {
	event := mb.Event{
		MetricSetFields: mapstr.M{},
	}
	measurements := make(map[string]float64, 0)
	for counterPath, values := range counters {
		if hasCounter, readerCounter := re.getCounter(counterPath); hasCounter {
			for _, val := range values {
				// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
				// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
				if val.Err.Error != nil {
					if val.Err.Error == pdh.PDH_CALC_NEGATIVE_VALUE || val.Err.Error == pdh.PDH_INVALID_DATA {
						re.log.Debugw("Counter value retrieval returned",
							"error", val.Err.Error, "cstatus", pdh.PdhErrno(val.Err.CStatus), logp.Namespace("perfmon"), "query", counterPath)
						continue
					}
				}
				if val.Measurement == nil {
					continue
				}
				var counterVal float64
				switch val.Measurement.(type) {
				case int64:
					counterVal = float64(val.Measurement.(int64))
				case int:
					counterVal = float64(val.Measurement.(int))
				default:
					counterVal = val.Measurement.(float64)
				}
				if _, ok := measurements[readerCounter.QueryField]; !ok {
					measurements[readerCounter.QueryField] = counterVal
					measurements[readerCounter.QueryField+instanceCountLabel] = 1
				} else {
					measurements[readerCounter.QueryField+instanceCountLabel] = measurements[readerCounter.QueryField+instanceCountLabel] + 1
					measurements[readerCounter.QueryField] = measurements[readerCounter.QueryField] + counterVal
				}
			}
		}
	}
	for key, val := range measurements {
		if strings.Contains(key, instanceCountLabel) {
			if val == 1 {
				continue
			} else {
				event.MetricSetFields.Put(fmt.Sprintf("%s.%s", strings.Split(key, ".")[0], re.config.GroupAllCountersTo), val)
			}
		} else {
			event.MetricSetFields.Put(key, val)
		}
	}
	return event
}

// matchParentProcess will try to get the parent process name
func matchesParentProcess(instanceName string) (bool, string) {
	matches := processRegexp.FindStringSubmatch(instanceName)
	if len(matches) == 2 {
		return true, matches[1]
	}
	return false, instanceName
}
