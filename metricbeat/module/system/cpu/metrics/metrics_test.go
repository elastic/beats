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

package metrics

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestCPUGet(t *testing.T) {
	root := ""
	if runtime.GOOS == "freebsd" {
		root = "/compat/linux/proc/"
	}
	metrics, err := Get(root)

	assert.NoError(t, err, "error in Get")
	assert.NotZero(t, metrics.Total(), "got total zero")

	time.Sleep(time.Second * 10)

	secondMetrics, err := Get(root)
	assert.NoError(t, err, "error in Get")
	assert.NotZero(t, metrics.Total(), "got total zero")

	events := common.MapStr{}
	secondMetrics.FillPercentages(&events, metrics)

	total, err := events.GetValue("total.pct")
	assert.NoError(t, err, "error finding total.pct")
	assert.NotZero(t, total.(float64), "total is zero")

	secondMetrics.FillNormalizedPercentages(&events, metrics)

	totalNorm, err := events.GetValue("total.norm.pct")
	assert.NoError(t, err, "error finding total.pct")
	assert.NotZero(t, totalNorm.(float64), "total is zero")

	secondMetrics.FillTicks(&events)

	t.Logf("Got metrics: \n%s", events.StringToPrint())
}
