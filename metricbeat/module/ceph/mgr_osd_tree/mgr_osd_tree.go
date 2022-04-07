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

package mgr_osd_tree

import (
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
	"github.com/elastic/beats/v8/metricbeat/module/ceph/mgr"
)

const (
	defaultScheme      = "https"
	defaultPath        = "/request"
	defaultQueryParams = "wait=1"

	cephPrefix = "osd tree"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		QueryParams:   defaultQueryParams,
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("ceph", "mgr_osd_tree", New,
		mb.WithHostParser(hostParser),
	)
}

type MetricSet struct {
	*mgr.MetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := mgr.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	metricSet = metricSet.WithPrefix(cephPrefix)
	return &MetricSet{metricSet}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	events, err := eventsMapping(content)
	if err != nil {
		return err
	}

	for _, event := range events {
		reported := reporter.Event(mb.Event{
			ModuleFields: common.MapStr{
				"osd_tree": event,
			}})
		if !reported {
			return nil
		}
	}
	return nil
}
