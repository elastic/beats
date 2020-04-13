// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build googlecloud

package storage

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud"
)

func TestData(t *testing.T) {
	config := googlecloud.GetConfigForTest(t, "storage")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
