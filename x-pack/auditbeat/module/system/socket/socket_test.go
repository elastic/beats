// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package socket

import (
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
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

	// The first socket (events[0]) is usually something like rpcbind,
	// the last one should be more interesting.
	fullEvent := mbtest.StandardizeEvent(f, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func TestFetch(t *testing.T) {
	// Consume first event: list of all currently open sockets
	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, errs := mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}
	_, err := events[0].RootFields.HasKey("destination.port")
	assert.NoError(t, err)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	i := strings.LastIndex(addr, ":")
	listenerPort, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		t.Fatal("failed to get port from addr", addr)
	}

	// Consume second event: Socket we just opened
	events, errs = mbtest.ReportingFetchV2(ms)
	if errs != nil {
		t.Fatal("fetch", errs)
	}

	var found bool
	for _, evt := range events {
		port, ok := getRequiredValue("destination.port", evt, t).(int)
		if !ok {
			t.Fatal("destination.port is not an int")
		}
		if port != listenerPort {
			continue
		}

		pid, ok := getRequiredValue("process.pid", evt, t).(int)
		if !ok {
			t.Fatal("process.pid is not an int")
		}
		assert.Equal(t, os.Getpid(), pid)

		processName, ok := getRequiredValue("process.name", evt, t).(string)
		if !ok {
			t.Fatal("process.name is not a string")
		}
		assert.Equal(t, "socket.test", processName)

		uid, ok := getRequiredValue("user.id", evt, t).(uint32)
		if !ok {
			t.Fatal("user.uid is not a uint32")
		}
		assert.EqualValues(t, os.Geteuid(), uid)

		dir, ok := getRequiredValue("network.direction", evt, t).(string)
		if !ok {
			t.Fatal("network.direction is not a string")
		}
		assert.Equal(t, "listening", dir)

		found = true
		break
	}

	assert.True(t, found, "listener not found")
}

func getRequiredValue(key string, mbEvent mb.Event, t testing.TB) interface{} {
	v, err := mbEvent.RootFields.GetValue(key)
	if err != nil {
		t.Fatalf("err=%v, key=%v, event=%v", key, err, mbEvent)
	}
	if v == nil {
		t.Fatalf("key %v not found in %v", key, mbEvent)
	}
	return v
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"socket"},
	}
}
