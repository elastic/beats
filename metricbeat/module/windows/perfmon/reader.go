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
	"strings"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

const instanceCountLabel = ":count"

// Reader will contain the config options
type Reader struct {
	query    pdh.Query    // PDH Query
	executed bool         // Indicates if the query has been executed.
	log      *logp.Logger //
	config   Config       // Metricset configuration
}

// NewReader creates a new instance of Reader.
func NewReader(config Config) (*Reader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	r := &Reader{
		query: query,
		log:   logp.NewLogger("perfmon"),
	}
	for i, counter := range config.Counters {
		config.Counters[i].ChildQueries = []string{}
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
			config.Counters[i].ChildQueries = append(config.Counters[i].ChildQueries, v)
		}
	}
	r.config = config
	return r, nil
}

// RefreshCounterPaths will recheck for any new instances and add them to the counter list
func (re *Reader) RefreshCounterPaths() error {
	var newCounters []string
	for i, counter := range re.config.Counters {
		re.config.Counters[i].ChildQueries = []string{}
		childQueries, err := re.query.GetCounterPaths(counter.Query)
		if err != nil {
			if re.config.IgnoreNECounters {
				switch err {
				case pdh.PDH_CSTATUS_NO_COUNTER, pdh.PDH_CSTATUS_NO_COUNTERNAME,
					pdh.PDH_CSTATUS_NO_INSTANCE, pdh.PDH_CSTATUS_NO_OBJECT:
					re.log.Infow("Ignoring non existent counter", "error", err,
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
				if err := re.query.AddCounter(v, counter.InstanceName, counter.Format, len(childQueries) > 1); err != nil {
					return errors.Wrapf(err, "failed to add counter (query='%v')", counter.Query)
				}
				re.config.Counters[i].ChildQueries = append(re.config.Counters[i].ChildQueries, v)
			}
		}
	}
	err := re.query.RemoveUnusedCounters(newCounters)
	if err != nil {
		return errors.Wrap(err, "failed removing unused counter values")
	}

	return nil
}

// Read executes a query and returns those values in an event.
func (re *Reader) Read() ([]mb.Event, error) {
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := re.query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := re.query.GetFormattedCounterValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}
	var events []mb.Event
	// GroupAllCountersTo config option where counters for all instances are aggregated and instance count is added in the event under the string value provided by this option.
	if re.config.GroupAllCountersTo != "" {
		event := re.groupToEvent(values)
		events = append(events, event)
	} else {
		events = re.groupToEvents(values)
	}
	re.executed = true
	return events, nil
}

// Close will close the PDH query for now.
func (re *Reader) Close() error {
	return re.query.Close()
}
