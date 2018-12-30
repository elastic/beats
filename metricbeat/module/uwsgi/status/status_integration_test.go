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

package status

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/uwsgi"

	"github.com/stretchr/testify/assert"
)

func TestFetchTCP(t *testing.T) {
	compose.EnsureUp(t, "uwsgi_tcp")

	f := mbtest.NewEventsFetcher(t, getConfig("tcp"))
	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.True(t, len(events) > 0)
	totals := findItems(events, "total")
	assert.Equal(t, 1, len(totals))
}

func TestFetchHTTP(t *testing.T) {
	compose.EnsureUp(t, "uwsgi_http")

	f := mbtest.NewEventsFetcher(t, getConfig("http"))
	events, err := f.Fetch()
	assert.NoError(t, err)

	assert.True(t, len(events) > 0)
	totals := findItems(events, "total")
	assert.Equal(t, 1, len(totals))
}

func getConfig(scheme string) map[string]interface{} {
	conf := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
	}

	switch scheme {
	case "tcp":
		conf["hosts"] = []string{uwsgi.GetEnvTCPServer()}
	case "http", "https":
		conf["hosts"] = []string{uwsgi.GetEnvHTTPServer()}
	default:
		conf["hosts"] = []string{uwsgi.GetEnvTCPServer()}
	}
	return conf
}
