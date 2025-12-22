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

//go:build integration

package keyspace

import (
	"strconv"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	rd "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	addEntry(t, service.Host())

	// Fetch data
	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, err := mbtest.ReportingFetchV2Error(ms)
	if err != nil {
		t.Fatal("fetch", err)
	}

	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), events)

	// Make sure at least 1 db keyspace exists
	assert.True(t, len(events) > 0)

	keyspace := events[0].MetricSetFields

	avgTTL, ok := keyspace["avg_ttl"].(int64)
	if !ok {
		t.Errorf("avg_ttl is not of type int64")
	}
	expires, ok := keyspace["expires"].(int64)
	if !ok {
		t.Errorf("expires is not of type int64")
	}
	keys, ok := keyspace["keys"].(int64)
	if !ok {
		t.Errorf("keys is not of type int64")
	}
	subExpiry, ok := keyspace["subexpiry"].(int64)
	if !ok {
		t.Errorf("subexpiry is not of type int64")
	}
	id, ok := keyspace["id"].(string)
	if !ok {
		t.Errorf("id is not of type string")
	}

	assert.True(t, avgTTL >= 0)
	assert.True(t, expires >= 0)
	assert.True(t, keys >= 0)
	assert.True(t, subExpiry >= 0)
	assert.True(t, strings.Contains(id, "db"))
}

func TestSubExpiryField(t *testing.T) {
	service := compose.EnsureUp(t, "redis")
	// Check redis version
	redisMajorVersion, redisMinorVersion := getRedisVersion(t, service.Host())
	if redisMajorVersion < 7 || (redisMajorVersion == 7 && redisMinorVersion < 4) {
		t.Skip("subexpiry field is only available in Redis version 7.4.0 and above")
	}
	addEntryWithExpiry(t, service.Host())

	// Fetch data
	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errSlice := mbtest.ReportingFetchV2Error(ms)
	if errSlice != nil {
		t.Fatal("fetch", errSlice)
	}

	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), events)

	// Make sure at least 1 db keyspace exists
	assert.True(t, len(events) > 0)

	var keyspace map[string]interface{}
	for _, event := range events {
		if id, ok := event.MetricSetFields["id"].(string); ok && id == "db0" {
			keyspace = event.MetricSetFields
		}
	}
	if keyspace == nil {
		t.Fatal("db0 keyspace not found in events")
	}

	subExpiry, ok := keyspace["subexpiry"].(int64)
	if !ok {
		t.Errorf("subexpiry is not of type int64")
	}
	assert.True(t, subExpiry > 0)
}

func getRedisVersion(t *testing.T, host string) (int, int) {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer c.Close()
	info, err := rd.String(c.Do("INFO", "server"))
	if err != nil {
		t.Fatal("INFO", err)
	}
	var version string
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "redis_version:") {
			version = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			break
		}
	}
	versionParts := strings.SplitN(version, ".", 3)
	redisMajorVersion, err := strconv.Atoi(versionParts[0])
	if err != nil {
		t.Fatalf("failed to parse redis version: %v", err)
	}
	redisMinorVersion, err := strconv.Atoi(versionParts[1])
	if err != nil {
		t.Fatalf("failed to parse redis version: %v", err)
	}
	return redisMajorVersion, redisMinorVersion
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	addEntry(t, service.Host())

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	err := mbtest.WriteEventsReporterV2Error(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

// addEntryWithExpiry adds an entry with field expiry to redis
func addEntryWithExpiry(t *testing.T, host string) {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer c.Close()
	_, err = c.Do("SELECT", 0)
	if err != nil {
		t.Fatal("SELECT", err)
	}
	_, err = c.Do("HSET", "test_hash", "f1", "v1", "f2", "v2")
	if err != nil {
		t.Fatal("HSET", err)
	}
	_, err = c.Do("HEXPIRE", "test_hash", 60, "FIELDS", 1, "f1")
	if err != nil {
		t.Fatal("HEXPIRE", err)
	}
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
