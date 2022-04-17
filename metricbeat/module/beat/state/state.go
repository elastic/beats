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

package state

import (
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	"github.com/menderesk/beats/v7/metricbeat/module/beat"
)

func init() {
	mb.Registry.MustAddMetricSet(beat.ModuleName, "state", New,
		mb.WithHostParser(hostParser),
	)
}

const (
	statePath = "state"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		DefaultPath:   statePath,
	}.Build()
)

// MetricSet defines all fields of the MetricSet
type MetricSet struct {
	*beat.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := beat.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	info, err := beat.GetInfo(m.MetricSet)
	if err != nil {
		return err
	}

	return eventMapping(r, *info, content, m.XPackEnabled)
}
