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

package queue

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/rabbitmq/mtest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	server := mtest.Server(t, mtest.DefaultServerConfig)
	defer server.Close()

	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	err := metricSet.Fetch(reporter)
	assert.NoError(t, err)

	e := mbtest.StandardizeEvent(metricSet, reporter.GetEvents()[0])
	t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())

	ee, _ := e.Fields.GetValue("rabbitmq.queue")
	event := ee.(mapstr.M)

	assert.EqualValues(t, "queuenamehere", event["name"])
	assert.EqualValues(t, true, event["durable"])
	assert.EqualValues(t, false, event["auto_delete"])
	assert.EqualValues(t, false, event["exclusive"])
	assert.EqualValues(t, "running", event["state"])

	arguments := event["arguments"].(mapstr.M)
	assert.EqualValues(t, 9, arguments["max_priority"])

	consumers := event["consumers"].(mapstr.M)
	utilisation := consumers["utilisation"].(mapstr.M)
	assert.EqualValues(t, 3, consumers["count"])
	assert.Equal(t, 0.7, utilisation["pct"])

	memory := event["memory"].(mapstr.M)
	assert.EqualValues(t, 232720, memory["bytes"])

	messages := event["messages"].(mapstr.M)
	total := messages["total"].(mapstr.M)
	ready := messages["ready"].(mapstr.M)
	unacknowledged := messages["unacknowledged"].(mapstr.M)
	persistent := messages["persistent"].(mapstr.M)
	assert.EqualValues(t, 74, total["count"])
	assert.EqualValues(t, 71, ready["count"])
	assert.EqualValues(t, 3, unacknowledged["count"])
	assert.EqualValues(t, 73, persistent["count"])

	totalDetails := total["details"].(mapstr.M)
	assert.EqualValues(t, 2.2, totalDetails["rate"])

	readyDetails := ready["details"].(mapstr.M)
	assert.EqualValues(t, 0, readyDetails["rate"])

	unacknowledgedDetails := unacknowledged["details"].(mapstr.M)
	assert.EqualValues(t, 0.5, unacknowledgedDetails["rate"])

	disk := event["disk"].(mapstr.M)
	reads := disk["reads"].(mapstr.M)
	writes := disk["writes"].(mapstr.M)
	assert.EqualValues(t, 212, reads["count"])
	assert.EqualValues(t, 121, writes["count"])
}

func TestData(t *testing.T) {
	server := mtest.Server(t, mtest.DefaultServerConfig)
	defer server.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	err := mbtest.WriteEventsReporterV2ErrorCond(ms, t, "", func(e mapstr.M) bool {
		hasTotal, _ := e.HasKey("rabbitmq.queue.messages.total")
		return hasTotal
	})
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(url string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"queue"},
		"hosts":      []string{url},
	}
}
