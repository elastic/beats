// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"os/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/go-sysinfo/types"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	// The first process (events[0]) is usually something like systemd,
	// the last one should be more interesting.
	fullEvent := mbtest.StandardizeEvent(f, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"process"},
	}
}

func TestProcessEvent(t *testing.T) {
	process := Process{
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
			UID:  "1002",
			EUID: "1002",
			SUID: "1002",
			GID:  "1002",
			EGID: "1002",
			SGID: "1002",
		},
		User: &user.User{
			Uid:      "1002",
			Username: "elastic",
		},
		Group: &user.Group{
			Gid:  "1002",
			Name: "elastic",
		},
	}
	eventType := eventTypeEvent
	eventAction := eventActionProcessStarted

	event := processEvent(&process, eventType, eventAction)

	containsError, err := event.RootFields.HasKey("error")
	if assert.NoError(t, err) {
		assert.False(t, containsError)
	}

	expectedRootFields := map[string]interface{}{
		"event.kind":   "event",
		"event.action": "process_started",
		"message":      "Process zsh (PID: 9086) by user elastic STARTED",

		"process.pid":        9086,
		"process.ppid":       9085,
		"process.name":       "zsh",
		"process.executable": "/bin/zsh",
		"process.args":       []string{"zsh"},
		"process.start":      "2019-01-01 00:00:01 +0000 UTC",

		"user.id":                 "1002",
		"user.name":               "elastic",
		"user.group.id":           "1002",
		"user.group.name":         "elastic",
		"user.effective.id":       "1002",
		"user.effective.group.id": "1002",
		"user.saved.id":           "1002",
		"user.saved.group.id":     "1002",
	}
	for expFieldName, expFieldValue := range expectedRootFields {
		value, err := event.RootFields.GetValue(expFieldName)
		if assert.NoErrorf(t, err, "error for field %v (value: %v)", expFieldName, expFieldValue) {
			switch v := value.(type) {
			case time.Time:
				assert.Equalf(t, expFieldValue, v.String(), "Unexpected value for field %v.", expFieldName)
			default:
				assert.Equalf(t, expFieldValue, value, "Unexpected value for field %v.", expFieldName)
			}
		}
	}
}

func TestPutIfNotEmpty(t *testing.T) {
	mapstr := common.MapStr{}

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
