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
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/metricbeat/mb/parse"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

const metricsetName = "perfmon"

func init() {
	mb.Registry.MustAddMetricSet("windows", metricsetName, New, mb.WithHostParser(parse.EmptyHostParser))
}

type MetricSet struct {
	mb.BaseMetricSet
	reader *Reader
	log    *logp.Logger
}

// New create a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	reader, err := NewReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "initialization of reader failed")
	}
	return &MetricSet{
		BaseMetricSet: base,
		reader:        reader,
		log:           logp.NewLogger(metricsetName),
	}, nil
}

// Fetch fetches events and reports them upstream
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// if the ignore_non_existent_counters flag is set and no valid counter paths are found the Read func will still execute, a check is done before
	if len(m.reader.query.Counters) == 0 {
		m.log.Error("no counter paths were found")
	}
	// refresh performance counter list
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if m.reader.config.RefreshWildcardCounters {
		err := m.reader.RefreshCounterPaths()
		if err != nil {
			return errors.Wrap(err, "failed retrieving counters")
		}
	}
	events, err := m.reader.Read()
	if err != nil {
		return errors.Wrap(err, "failed reading counters")
	}

	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}

	return nil
}

// Close will be called when metricbeat is stopped, should close the query.
func (m *MetricSet) Close() error {
	err := m.reader.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close pdh query")
	}
	return nil
}
