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

package node

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/rabbitmq/mtest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
)

func TestFetchNodeEventContents(t *testing.T) {
	testFetch(t, configCollectNode)
}

func TestFetchClusterEventContents(t *testing.T) {
	testFetch(t, configCollectCluster)
}

func testFetch(t *testing.T, collect string) {
	server := mtest.Server(t, mtest.DefaultServerConfig)
	defer server.Close()

	config := map[string]interface{}{
		"module":       "rabbitmq",
		"metricsets":   []string{"node"},
		"hosts":        []string{server.URL},
		"node.collect": collect,
	}

	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errors := mbtest.ReportingFetchV2Error(ms)
	if !assert.True(t, len(errors) == 0, "There shouldn't be errors") {
		t.Log(errors)
	}
	if !assert.True(t, len(events) > 0, "There should be events") {
		t.FailNow()
	}
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), event.StringToPrint())

	disk := event["disk"].(mapstr.M)
	free := disk["free"].(mapstr.M)
	assert.EqualValues(t, int64(98317942784), free["bytes"])

	limit := free["limit"].(mapstr.M)
	assert.EqualValues(t, 50000000, limit["bytes"])

	fd := event["fd"].(mapstr.M)
	assert.EqualValues(t, 1048576, fd["total"])
	assert.EqualValues(t, 31, fd["used"])

	gc := event["gc"].(mapstr.M)
	num := gc["num"].(mapstr.M)
	assert.EqualValues(t, 1049055, num["count"])
	reclaimed := gc["reclaimed"].(mapstr.M)
	assert.EqualValues(t, int64(27352751800), reclaimed["bytes"])

	io := event["io"].(mapstr.M)
	fileHandle := io["file_handle"].(mapstr.M)
	openAttempt := fileHandle["open_attempt"].(mapstr.M)
	avg := openAttempt["avg"].(mapstr.M)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 597670, openAttempt["count"])

	read := io["read"].(mapstr.M)
	avg = read["avg"].(mapstr.M)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 1, read["bytes"])
	assert.EqualValues(t, 3, read["count"])

	reopen := io["reopen"].(mapstr.M)
	assert.EqualValues(t, 3, reopen["count"])

	seek := io["seek"].(mapstr.M)
	avg = seek["avg"].(mapstr.M)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 23, seek["count"])

	sync := io["sync"].(mapstr.M)
	avg = sync["avg"].(mapstr.M)
	assert.EqualValues(t, 2, avg["ms"])
	assert.EqualValues(t, 149402, sync["count"])

	write := io["write"].(mapstr.M)
	avg = write["avg"].(mapstr.M)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 36305460, write["bytes"])
	assert.EqualValues(t, 149402, write["count"])

	mem := event["mem"].(mapstr.M)
	limit = mem["limit"].(mapstr.M)
	assert.EqualValues(t, int64(6628692787), limit["bytes"])
	used := mem["used"].(mapstr.M)
	assert.EqualValues(t, 105504768, used["bytes"])

	mnesia := event["mnesia"].(mapstr.M)
	disk = mnesia["disk"].(mapstr.M)
	tx := disk["tx"].(mapstr.M)
	assert.EqualValues(t, 1, tx["count"])
	ram := mnesia["ram"].(mapstr.M)
	tx = ram["tx"].(mapstr.M)
	assert.EqualValues(t, 92, tx["count"])

	msg := event["msg"].(mapstr.M)
	storeRead := msg["store_read"].(mapstr.M)
	assert.EqualValues(t, 0, storeRead["count"])
	storeWrite := msg["store_write"].(mapstr.M)
	assert.EqualValues(t, 0, storeWrite["count"])

	assert.EqualValues(t, "rabbit@e2b1ae6390fd", event["name"])

	proc := event["proc"].(mapstr.M)
	assert.EqualValues(t, 1048576, proc["total"])
	assert.EqualValues(t, 403, proc["used"])

	assert.EqualValues(t, 4, event["processors"])

	queue := event["queue"].(mapstr.M)
	index := queue["index"].(mapstr.M)
	journalWrite := index["journal_write"].(mapstr.M)
	assert.EqualValues(t, 448230, journalWrite["count"])
	read = index["read"].(mapstr.M)
	assert.EqualValues(t, 0, read["count"])
	write = index["write"].(mapstr.M)
	assert.EqualValues(t, 2, write["count"])

	run := event["run"].(mapstr.M)
	assert.EqualValues(t, 0, run["queue"])

	socket := event["socket"].(mapstr.M)
	assert.EqualValues(t, 943626, socket["total"])
	assert.EqualValues(t, 3, socket["used"])

	assert.EqualValues(t, "disc", event["type"])

	assert.EqualValues(t, 98754834, event["uptime"])
}
