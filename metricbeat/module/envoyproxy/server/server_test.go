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

package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

const testFile = "../_meta/test/serverstats"

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("../_meta/test/serverstats")
	assert.NoError(t, err)

	event, err := eventMapping(content)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, event, 7, "got wrong number of event")

	clusterManager := event["cluster_manager"].(common.MapStr)
	assert.Equal(t, int64(1), clusterManager["active_clusters"])
	assert.Equal(t, int64(1), clusterManager["cluster_added"])
	assert.Equal(t, int64(0), clusterManager["cluster_modified"])
	assert.Equal(t, int64(0), clusterManager["cluster_removed"])
	assert.Equal(t, int64(0), clusterManager["warming_clusters"])

	fileSystem := event["filesystem"].(common.MapStr)
	assert.Equal(t, int64(389), fileSystem["flushed_by_timer"])
	assert.Equal(t, int64(0), fileSystem["reopen_failed"])
	assert.Equal(t, int64(44), fileSystem["write_buffered"])
	assert.Equal(t, int64(43), fileSystem["write_completed"])
	assert.Equal(t, int64(0), fileSystem["write_total_buffered"])

	server := event["server"].(common.MapStr)
	assert.Equal(t, int64(2147483647), server["days_until_first_cert_expiring"])
	assert.Equal(t, int64(1), server["live"])
	assert.Equal(t, int64(3120760), server["memory_allocated"])
	assert.Equal(t, int64(4194304), server["memory_heap_size"])
	assert.Equal(t, int64(0), server["parent_connections"])
	assert.Equal(t, int64(0), server["total_connections"])
	assert.Equal(t, int64(5025), server["uptime"])
	assert.Equal(t, int64(16364036), server["version"])
	assert.Equal(t, int64(4), server["watchdog_mega_miss"])
	assert.Equal(t, int64(4), server["watchdog_miss"])

	stats := event["stats"].(common.MapStr)
	assert.Equal(t, int64(0), stats["overflow"])
}

func TestFetchEventContent(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/serverstats")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "envoyproxy",
		"metricsets": []string{"server"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func testValue(t *testing.T, event common.MapStr, field string, value interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, data, value, "Wrong value for field "+field)
}

func TestFetchTimeout(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/serverstats")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Write([]byte(response))
		<-r.Context().Done()
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "envoyproxy",
		"metricsets": []string{"server"},
		"hosts":      []string{server.URL},
		"timeout":    "50ms",
	}

	f := mbtest.NewEventFetcher(t, config)

	start := time.Now()
	_, err = f.Fetch()
	elapsed := time.Since(start)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "request canceled (Client.Timeout exceeded")
	}

	assert.True(t, elapsed < 5*time.Second, "elapsed time: %s", elapsed.String())
}
