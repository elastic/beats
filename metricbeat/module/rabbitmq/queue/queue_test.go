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

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"

	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	server := mtest.Server(t, mtest.DefaultServerConfig)
	defer server.Close()

	config := map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"queue"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "queuenamehere", event["name"])
	assert.EqualValues(t, "/", event["vhost"])
	assert.EqualValues(t, true, event["durable"])
	assert.EqualValues(t, false, event["auto_delete"])
	assert.EqualValues(t, false, event["exclusive"])
	assert.EqualValues(t, "running", event["state"])
	assert.EqualValues(t, "rabbit@localhost", event["node"])

	arguments := event["arguments"].(common.MapStr)
	assert.EqualValues(t, 9, arguments["max_priority"])

	consumers := event["consumers"].(common.MapStr)
	utilisation := consumers["utilisation"].(common.MapStr)
	assert.EqualValues(t, 3, consumers["count"])
	assert.EqualValues(t, 0.7, utilisation["pct"])

	memory := event["memory"].(common.MapStr)
	assert.EqualValues(t, 232720, memory["bytes"])

	messages := event["messages"].(common.MapStr)
	total := messages["total"].(common.MapStr)
	ready := messages["ready"].(common.MapStr)
	unacknowledged := messages["unacknowledged"].(common.MapStr)
	persistent := messages["persistent"].(common.MapStr)
	assert.EqualValues(t, 74, total["count"])
	assert.EqualValues(t, 71, ready["count"])
	assert.EqualValues(t, 3, unacknowledged["count"])
	assert.EqualValues(t, 73, persistent["count"])

	totalDetails := total["details"].(common.MapStr)
	assert.EqualValues(t, 2.2, totalDetails["rate"])

	readyDetails := ready["details"].(common.MapStr)
	assert.EqualValues(t, 0, readyDetails["rate"])

	unacknowledgedDetails := unacknowledged["details"].(common.MapStr)
	assert.EqualValues(t, 0.5, unacknowledgedDetails["rate"])

	disk := event["disk"].(common.MapStr)
	reads := disk["reads"].(common.MapStr)
	writes := disk["writes"].(common.MapStr)
	assert.EqualValues(t, 212, reads["count"])
	assert.EqualValues(t, 121, writes["count"])
}
