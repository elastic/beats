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

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	logp.TestingSetup()

	service := compose.EnsureUp(t, "etcd")

	m := mbtest.NewFetcher(t, getConfig(service.Host()))
	events, errs := m.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	t.Logf("%s/%s event: %+v", m.Module().Name(), m.Name(), events[0])
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "etcd")

	m := mbtest.NewFetcher(t, getConfig(service.Host()))
	m.WriteEvents(t, "")
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "etcd",
		"metricsets": []string{"metrics"},
		"hosts":      []string{host},
	}
}
