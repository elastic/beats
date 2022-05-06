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

package mntr

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func assertExpectations(t *testing.T, expectations mapstr.M, report mapstr.M, message ...string) {
	for key, expectation := range expectations {
		assert.Contains(t, report, key, message)
		switch expectation := expectation.(type) {
		case mapstr.M:
			nestedReport, _ := report.GetValue(key)
			assert.IsType(t, nestedReport, report, message)
			assertExpectations(t, expectation, nestedReport.(mapstr.M), message...)
		default:
			reportValue, _ := report.GetValue(key)
			assert.Equal(t, expectation, reportValue, message)
		}
	}
}

//go:embed testdata/mntr.35.leader.txt
var mntrTestInputZooKeeper35 string

//go:embed testdata/mntr.37.leader.txt
var mntrTestInputZooKeeper37 string

func TestEventMapping(t *testing.T) {

	type TestCase struct {
		Version        string
		MntrSample     string
		ExpectedValues mapstr.M
	}

	mntrSamples := []TestCase{
		{
			"3.5.3",
			mntrTestInputZooKeeper35,
			mapstr.M{
				"learners":  int64(1),
				"followers": int64(1),
				"latency": mapstr.M{
					"max": float64(29),
					"avg": float64(0),
					"min": float64(0),
				},
			},
		},
		{
			"3.7.0",
			mntrTestInputZooKeeper37,
			mapstr.M{
				"learners":  int64(1),
				"followers": int64(1),
				"latency": mapstr.M{
					"max": float64(8),
					"avg": float64(0.5714),
					"min": float64(0),
				},
			},
		},
	}

	logger := logp.NewLogger("mntr_test")

	for i, sample := range mntrSamples {
		t.Run(sample.Version, func(t *testing.T) {

			reporter := &mbtest.CapturingReporterV2{}

			eventMapping(fmt.Sprint(i), bytes.NewReader([]byte(sample.MntrSample)), reporter, logger)

			assert.Empty(t, reporter.GetErrors())

			events := reporter.GetEvents()
			assert.Len(t, events, 1)

			event := events[len(events)-1]

			assertExpectations(t, sample.ExpectedValues, event.MetricSetFields)
		})
	}

}
