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

	"github.com/elastic/beats/v8/libbeat/common"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/metricbeat/module/rabbitmq/mtest"

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

	disk := event["disk"].(common.MapStr)
	free := disk["free"].(common.MapStr)
	assert.EqualValues(t, int64(98317942784), free["bytes"])

	limit := free["limit"].(common.MapStr)
	assert.EqualValues(t, 50000000, limit["bytes"])

	fd := event["fd"].(common.MapStr)
	assert.EqualValues(t, 1048576, fd["total"])
	assert.EqualValues(t, 31, fd["used"])

	gc := event["gc"].(common.MapStr)
	num := gc["num"].(common.MapStr)
	assert.EqualValues(t, 1049055, num["count"])
	reclaimed := gc["reclaimed"].(common.MapStr)
	assert.EqualValues(t, int64(27352751800), reclaimed["bytes"])

	io := event["io"].(common.MapStr)
	fileHandle := io["file_handle"].(common.MapStr)
	openAttempt := fileHandle["open_attempt"].(common.MapStr)
	avg := openAttempt["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 597670, openAttempt["count"])

	read := io["read"].(common.MapStr)
	avg = read["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 1, read["bytes"])
	assert.EqualValues(t, 3, read["count"])

	reopen := io["reopen"].(common.MapStr)
	assert.EqualValues(t, 3, reopen["count"])

	seek := io["seek"].(common.MapStr)
	avg = seek["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 23, seek["count"])

	sync := io["sync"].(common.MapStr)
	avg = sync["avg"].(common.MapStr)
	assert.EqualValues(t, 2, avg["ms"])
	assert.EqualValues(t, 149402, sync["count"])

	write := io["write"].(common.MapStr)
	avg = write["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 36305460, write["bytes"])
	assert.EqualValues(t, 149402, write["count"])

	mem := event["mem"].(common.MapStr)
	limit = mem["limit"].(common.MapStr)
	assert.EqualValues(t, int64(6628692787), limit["bytes"])
	used := mem["used"].(common.MapStr)
	assert.EqualValues(t, 105504768, used["bytes"])

	mnesia := event["mnesia"].(common.MapStr)
	disk = mnesia["disk"].(common.MapStr)
	tx := disk["tx"].(common.MapStr)
	assert.EqualValues(t, 1, tx["count"])
	ram := mnesia["ram"].(common.MapStr)
	tx = ram["tx"].(common.MapStr)
	assert.EqualValues(t, 92, tx["count"])

	msg := event["msg"].(common.MapStr)
	storeRead := msg["store_read"].(common.MapStr)
	assert.EqualValues(t, 0, storeRead["count"])
	storeWrite := msg["store_write"].(common.MapStr)
	assert.EqualValues(t, 0, storeWrite["count"])

	assert.EqualValues(t, "rabbit@e2b1ae6390fd", event["name"])

	proc := event["proc"].(common.MapStr)
	assert.EqualValues(t, 1048576, proc["total"])
	assert.EqualValues(t, 403, proc["used"])

	assert.EqualValues(t, 4, event["processors"])

	queue := event["queue"].(common.MapStr)
	index := queue["index"].(common.MapStr)
	journalWrite := index["journal_write"].(common.MapStr)
	assert.EqualValues(t, 448230, journalWrite["count"])
	read = index["read"].(common.MapStr)
	assert.EqualValues(t, 0, read["count"])
	write = index["write"].(common.MapStr)
	assert.EqualValues(t, 2, write["count"])

	run := event["run"].(common.MapStr)
	assert.EqualValues(t, 0, run["queue"])

	socket := event["socket"].(common.MapStr)
	assert.EqualValues(t, 943626, socket["total"])
	assert.EqualValues(t, 3, socket["used"])

	assert.EqualValues(t, "disc", event["type"])

	assert.EqualValues(t, 98754834, event["uptime"])
}
