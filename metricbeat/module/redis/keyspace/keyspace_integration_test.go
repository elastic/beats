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

package keyspace

import (
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
)

var runner = compose.TestRunner{
	Service: "redis",
	Options: map[string][]string{
		"REDIS_VERSION": []string{
			"3.2.4",
			"4.0.11",
			"5.0-rc",
		},
		"IMAGE_OS": []string{
			"alpine",
			"stretch",
		},
	},
	Parallel: true,
}

func TestFetch(t *testing.T) {
	t.Parallel()

	runner.Run(t, func(t *testing.T, host string) {
		addEntry(t, host)

		// Fetch data
		f := mbtest.NewEventsFetcher(t, getConfig(host))
		events, err := f.Fetch()
		if err != nil {
			t.Fatal("fetch", err)
		}

		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events)

		// Make sure at least 1 db keyspace exists
		assert.True(t, len(events) > 0)

		keyspace := events[0]

		assert.True(t, keyspace["avg_ttl"].(int64) >= 0)
		assert.True(t, keyspace["expires"].(int64) >= 0)
		assert.True(t, keyspace["keys"].(int64) >= 0)
		assert.True(t, strings.Contains(keyspace["id"].(string), "db"))
	})
}

func TestData(t *testing.T) {
	t.Parallel()

	// TODO: Fix EnsureUp
	runner := compose.TestRunner{
		Service: "redis",
		Options: map[string][]string{
			"REDIS_VERSION": []string{
				"4.0.11",
			},
			"IMAGE_OS": []string{
				"alpine",
			},
		},
		Parallel: true,
	}
	runner.Run(t, func(t *testing.T, host string) {
		addEntry(t, host)

		f := mbtest.NewEventsFetcher(t, getConfig(host))

		err := mbtest.WriteEvents(f, t)
		if err != nil {
			t.Fatal("write", err)
		}
	})
}

// addEntry adds an entry to redis
func addEntry(t *testing.T, host string) {
	// Insert at least one event to make sure db exists
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer c.Close()
	_, err = c.Do("SET", "foo", "bar")
	if err != nil {
		t.Fatal("SET", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"keyspace"},
		"hosts":      []string{host},
	}
}
