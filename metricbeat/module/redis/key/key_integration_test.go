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

package key

import (
	"fmt"
	"testing"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	addEntry(t, service.Host(), "foo", 1)

	ms := mbtest.NewFetcher(t, getConfig(service.Host()))
	events, err := ms.FetchEvents()
	if err != nil {
		t.Fatal("fetch", err)
	}

	t.Logf("%s/%s events: %+v", ms.Module().Name(), ms.Name(), events)
	assert.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	addEntry(t, service.Host(), "foo", 1)

	ms := mbtest.NewFetcher(t, getConfig(service.Host()))
	ms.WriteEvents(t, "")
}

func TestFetchMultipleKeyspaces(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	expectedKeyspaces := map[string]uint{
		"foo": 0,
		"bar": 1,
		"baz": 2,
	}
	expectedEvents := len(expectedKeyspaces)

	for name, keyspace := range expectedKeyspaces {
		addEntry(t, service.Host(), name, keyspace)
	}

	config := getConfig(service.Host())
	config["key.patterns"] = []map[string]interface{}{
		{
			"pattern":  "foo",
			"keyspace": 0,
		},
		{
			"pattern": "bar",
			// keyspace set to 1 in the host url
		},
		{
			"pattern":  "baz",
			"keyspace": 2,
		},
	}

	ms := mbtest.NewFetcher(t, config)
	events, err := ms.FetchEvents()

	assert.Len(t, err, 0)
	assert.Len(t, events, expectedEvents)

	for _, event := range events {
		name := event.MetricSetFields["name"].(string)
		expectedKeyspace, found := expectedKeyspaces[name]
		if !assert.True(t, found, name+" not expected") {
			continue
		}
		id := event.MetricSetFields["id"].(string)
		assert.Equal(t, fmt.Sprintf("%d:%s", expectedKeyspace, name), id)
		keyspace := event.ModuleFields["keyspace"].(common.MapStr)
		keyspaceID := keyspace["id"].(string)
		assert.Equal(t, fmt.Sprintf("db%d", expectedKeyspace), keyspaceID)
	}
}

// addEntry adds an entry to redis
func addEntry(t *testing.T, host string, key string, keyspace uint) {
	// Insert at least one event to make sure db exists
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	_, err = c.Do("SELECT", keyspace)
	if err != nil {
		t.Fatal("select", err)
	}
	defer c.Close()
	_, err = c.Do("SET", key, "bar", "EX", "360")
	if err != nil {
		t.Fatal("SET", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"key"},
		"hosts":      []string{host + "/1"},
		"key.patterns": []map[string]interface{}{
			{
				"pattern": "foo",
			},
		},
	}
}
