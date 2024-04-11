// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"os/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/auditbeat/ab"
	"github.com/elastic/beats/v7/auditbeat/core"
	"github.com/elastic/beats/v7/auditbeat/helper/hasher"
	abtest "github.com/elastic/beats/v7/auditbeat/testing"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo/types"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	f := mbtest.NewReportingMetricSetV2WithRegistry(t, getConfig(), ab.Registry)

	// Set lastState and add test process to cache so it will be reported as stopped.
	f.(*MetricSet).lastState = time.Now()
	p := testProcess()
	f.(*MetricSet).cache.DiffAndUpdateCache(convertToCacheable([]*Process{p}))

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	fullEvent := mbtest.StandardizeEvent(f, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":   system.ModuleName,
		"datasets": []string{"process"},

		// To speed things up during testing, we effectively
		// disable hashing.
		"process.hash.max_file_size": 1,
	}
}

func TestProcessEvent(t *testing.T) {
	ms := mbtest.NewReportingMetricSetV2WithRegistry(t, getConfig(), ab.Registry).(*MetricSet)

	eventType := eventTypeEvent
	eventAction := eventActionProcessStarted
	event := ms.processEvent(testProcess(), eventType, eventAction)

	containsError, err := event.RootFields.HasKey("error")
	if assert.NoError(t, err) {
		assert.False(t, containsError)
	}

	expectedRootFields := map[string]interface{}{
		"event.kind":     "event",
		"event.category": []string{"process"},
		"event.type":     []string{"start"},
		"event.action":   "process_started",
		"message":        "Process zsh (PID: 9086) by user elastic STARTED",

		"process.pid":        9086,
		"process.parent.pid": 9085,
		"process.name":       "zsh",
		"process.executable": "/bin/zsh",
		"process.args":       []string{"zsh"},
		"process.start":      "2019-01-01 00:00:01 +0000 UTC",
		"process.hash.sha1":  "3de6a0a1cf514d15a61d3c873e2a710977c1103d",

		"user.id":                 "1000",
		"user.name":               "elastic",
		"user.group.id":           "1000",
		"user.group.name":         "elastic",
		"user.effective.id":       "1000",
		"user.effective.group.id": "1000",
		"user.saved.id":           "1000",
		"user.saved.group.id":     "1000",
	}
	for expFieldName, expFieldValue := range expectedRootFields {
		value, err := event.RootFields.GetValue(expFieldName)
		if assert.NoErrorf(t, err, "error for field %v (value: %v)", expFieldName, expFieldValue) {
			switch v := value.(type) {
			case time.Time:
				assert.Equalf(t, expFieldValue, v.String(), "Unexpected value for field %v.", expFieldName)
			case hasher.Digest:
				assert.Equalf(t, expFieldValue, string(v), "Unexpected value for field %v.", expFieldName)
			default:
				assert.Equalf(t, expFieldValue, value, "Unexpected value for field %v.", expFieldName)
			}
		}
	}
}

func testProcess() *Process {
	return &Process{
		Info: types.ProcessInfo{
			Name:      "zsh",
			PID:       9086,
			PPID:      9085,
			CWD:       "/home/elastic",
			Exe:       "/bin/zsh",
			Args:      []string{"zsh"},
			StartTime: time.Date(2019, 1, 1, 0, 0, 1, 0, time.UTC),
		},
		UserInfo: &types.UserInfo{
			UID:  "1000",
			EUID: "1000",
			SUID: "1000",
			GID:  "1000",
			EGID: "1000",
			SGID: "1000",
		},
		User: &user.User{
			Uid:      "1000",
			Username: "elastic",
		},
		Group: &user.Group{
			Gid:  "1000",
			Name: "elastic",
		},
		Hashes: map[hasher.HashType]hasher.Digest{
			hasher.SHA1: []byte("3de6a0a1cf514d15a61d3c873e2a710977c1103d"),
		},
	}
}

func TestPutIfNotEmpty(t *testing.T) {
	mapstr := mapstr.M{}

	putIfNotEmpty(&mapstr, "key1", "value")
	value, err := mapstr.GetValue("key1")
	if assert.NoError(t, err) {
		assert.Equal(t, "value", value)
	}

	putIfNotEmpty(&mapstr, "key2", "")
	hasKey, err := mapstr.HasKey("key2")
	if assert.NoError(t, err) {
		assert.False(t, hasKey)
	}
}
