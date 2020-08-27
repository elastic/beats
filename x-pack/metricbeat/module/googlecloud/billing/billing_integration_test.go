// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build googlecloud

package billing

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud/metrics"
)

func TestData(t *testing.T) {
	config := metrics.GetConfigForTest(t, "billing")
	config["period"] = "24h"
	config["dataset_id"] = "master_gcp"
	config["table_pattern"] = "gcp_billing_export_v1"
	config["cost_type"] = "regular"

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
