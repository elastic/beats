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

package event

import (
	"testing"
	"time"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/libbeat/tests/compose"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	ms := mbtest.NewPushMetricSetV2(t, getConfig())
	var events []mb.Event
	done := make(chan interface{})
	go func() {
		events = mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)
		close(done)
	}()

	compose.EnsureUp(t, "apache")
	<-done

	if len(events) == 0 {
		t.Fatal("received no events")
	}
	assertNoErrors(t, events)

	beatEvent := mbtest.StandardizeEvent(ms, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}

func assertNoErrors(t *testing.T, events []mb.Event) {
	t.Helper()

	for _, e := range events {
		t.Log(e)

		if e.Error != nil {
			t.Errorf("received error: %+v", e.Error)
		}
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "docker",
		"metricsets": []string{"event"},
		"hosts":      []string{"unix:///var/run/docker.sock"},
	}
}
