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

// +build integration

package partition

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kafka/mtest"
)

func TestPartition(t *testing.T) {
	t.Parallel()

	logp.TestingSetup(logp.WithSelectors("kafka"))

	mtest.Runner.Run(t, compose.Suite{
		"Data":  testData,
		"Topic": testTopic,
	})
}

func testData(t *testing.T, r compose.R) {
	mtest.GenerateKafkaData(t, "metricbeat-generate-data", r.Host())

	ms := mbtest.NewReportingMetricSetV2(t, getConfig("", r.Host()))
	err := mbtest.WriteEventsReporterV2(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func testTopic(t *testing.T, r compose.R) {
	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("test-metricbeat-%s", id)

	// Create initial topic
	mtest.GenerateKafkaData(t, testTopic, r.Host())

	f := mbtest.NewReportingMetricSetV2(t, getConfig(testTopic, r.Host()))
	dataBefore, err := mbtest.ReportingFetchV2(f)
	if err != nil {
		t.Fatal("write", err)
	}
	if len(dataBefore) == 0 {
		t.Errorf("No offsets fetched from topic (before): %v", testTopic)
	}
	t.Logf("before: %v", dataBefore)

	var n int64 = 10
	// Create n messages
	for i := int64(0); i < n; i++ {
		mtest.GenerateKafkaData(t, testTopic, r.Host())
	}

	dataAfter, err := mbtest.ReportingFetchV2(f)
	if err != nil {
		t.Fatal("write", err)
	}
	if len(dataAfter) == 0 {
		t.Errorf("No offsets fetched from topic (after): %v", testTopic)
	}
	t.Logf("after: %v", dataAfter)

	// Checks that no new topics / partitions were added
	assert.True(t, len(dataBefore) == len(dataAfter))

	var offsetBefore int64 = 0
	var offsetAfter int64 = 0

	// Its possible that other topics exists -> select the right data
	for _, data := range dataBefore {
		if data.ModuleFields["topic"].(common.MapStr)["name"] == testTopic {
			offsetBefore = data.MetricSetFields["offset"].(common.MapStr)["newest"].(int64)
		}
	}

	for _, data := range dataAfter {
		if data.ModuleFields["topic"].(common.MapStr)["name"] == testTopic {
			offsetAfter = data.MetricSetFields["offset"].(common.MapStr)["newest"].(int64)
		}
	}

	// Compares offset before and after
	if offsetBefore+n != offsetAfter {
		t.Errorf("Offset before: %v", offsetBefore)
		t.Errorf("Offset after: %v", offsetAfter)
	}
	assert.True(t, offsetBefore+n == offsetAfter)
}

func getConfig(topic string, host string) map[string]interface{} {
	var topics []string
	if topic != "" {
		topics = []string{topic}
	}

	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"partition"},
		"hosts":      []string{host},
		"topics":     topics,
	}
}
