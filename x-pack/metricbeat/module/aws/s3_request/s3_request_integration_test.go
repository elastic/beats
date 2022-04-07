// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws
// +build integration,aws

package s3_request

import (
	"testing"

	_ "github.com/elastic/beats/v8/libbeat/processors/actions"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/x-pack/metricbeat/module/aws/mtest"
)

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "s3_request", "60s")

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
