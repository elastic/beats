// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package elb

import (
	"os"
	"testing"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	xpackmb "github.com/elastic/beats/x-pack/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws/mtest"

	// Register input module and metricset
	_ "github.com/elastic/beats/x-pack/metricbeat/module/aws"
	_ "github.com/elastic/beats/x-pack/metricbeat/module/aws/cloudwatch"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(xpackmb.NewLightModulesSource("../../../module"))
}

func TestData(t *testing.T) {
	config, info := mtest.GetConfigForTest("aws", "elb", "300s")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}
