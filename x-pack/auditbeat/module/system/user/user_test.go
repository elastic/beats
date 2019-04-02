// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package user

import (
	"os/user"
	"testing"
	"time"

	"github.com/elastic/beats/auditbeat/core"
	abtest "github.com/elastic/beats/auditbeat/testing"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())

	// Set lastState and add test process to cache so it will be reported as stopped.
	f.(*MetricSet).lastState = time.Now()
	u := testUser()
	f.(*MetricSet).cache.DiffAndUpdateCache(convertToCacheable([]*User{u}))

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	for _, e := range events {
		if name, _ := e.RootFields.GetValue("user.name"); name == "elastic" {
			fullEvent := mbtest.StandardizeEvent(f, e, core.AddDatasetToEvent)
			mbtest.WriteEventToDataJSON(t, fullEvent, "")
			return
		}
	}

	t.Fatal("user not found")
}

func testUser() *User {
	return &User{
		Name: "elastic",
		UID:  "9999",
		GID:  "1001",
		Groups: []*user.Group{
			&user.Group{
				Gid:  "1001",
				Name: "elastic",
			},
			&user.Group{
				Gid:  "1002",
				Name: "docker",
			},
		},
		Dir: "/home/elastic",
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"user"},

		// Would require root access to /etc/shadow
		// which we usually don't have when testing.
		"user.detect_password_changes": false,
	}
}
