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

package mb_test

import (
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

var hostParser = parse.URLHostParserBuilder{
	DefaultScheme: "http",
}.Build()

func init() {
	// Register the MetricSetFactory function for the "status" MetricSet.
	mb.Registry.MustAddMetricSet("someapp", "status", NewMetricSet,
		mb.WithHostParser(hostParser),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
}

func NewMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	fmt.Println("someapp-status url=", base.HostData().SanitizedURI)
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch will be called periodically by the framework.
func (ms *MetricSet) Fetch(report mb.Reporter) {
	// Fetch data from the host at ms.HostData().URI and return the data.
	data, err := common.MapStr{
		"some_metric":          18.0,
		"answer_to_everything": 42,
	}, error(nil)
	if err != nil {
		// Report an error if it occurs.
		report.Error(err)
		return
	}

	// Otherwise report the collected data.
	report.Event(data)
}

// ExampleReportingMetricSet demonstrates how to register a MetricSetFactory
// and implement a ReportingMetricSet.
func ExampleReportingMetricSet() {}
