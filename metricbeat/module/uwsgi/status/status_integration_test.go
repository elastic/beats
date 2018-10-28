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

	"github.com/stretchr/testify/assert"
)

func testStatus(t *testing.T, service string) {
	runner := compose.TestRunner{Service: service}

	runner.Run(t, compose.Suite{
		"Fetch": func(t *testing.T, r compose.R) {
			f := mbtest.NewEventsFetcher(t, getConfig(t, service, r.Host()))
			events, err := f.Fetch()
			assert.NoError(t, err)

			assert.True(t, len(events) > 0)
			totals := findItems(events, "total")
			assert.Equal(t, 1, len(totals))
		},
		"Data": func(t *testing.T, r compose.R) {
			f := mbtest.NewEventsFetcher(t, getConfig(t, service, r.Host()))
			err := mbtest.WriteEvents(f, t)
			if err != nil {
				t.Fatal("write", err)
			}
		},
	})
}

func TestStatusTCP(t *testing.T) {
	testStatus(t, "uwsgi_tcp")
}

func TestStatusHTTP(t *testing.T) {
	testStatus(t, "uwsgi_http")
}

func getConfig(t *testing.T, service string, host string) map[string]interface{} {
	conf := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
	}

	switch service {
	case "uwsgi_tcp":
		conf["hosts"] = []string{"tcp://" + host}
	case "uwsgi_http":
		conf["hosts"] = []string{"http://" + host}
	default:
		t.Errorf("Unexpected service: %s", service)
	}
	return conf
}
