// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package integration_tests

import (
	"github.com/pulumi/pulumi/pkg/v2/testing/integration"
	"github.com/pulumi/pulumi/sdk/v2/go/common/apitype"
	"os"
	"path"

	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/test"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/storage"
)

const location          = "WestEurope"



func TestFetch(t *testing.T) {
	config :=  map[string]interface{}{
		"module":                "azure",
		"period":                "300s",
		"refresh_list_interval": "600s",
		"metricsets":            []string{"storage"},
		"client_id":             "26a2f804-b87d-4112-babd-e373dbf1e7a1",
		"client_secret":         "testtesttest",
		"tenant_id":             "aa40685b-417d-4664-b4ec-8f7640719adb",
		"subscription_id":       "70bd6e77-4b1e-4835-8896-db77b8eef364",
	}
	config["resources"] = []map[string]interface{}{{
		"resource_id": "jhj",
		"metrics": []map[string]interface{}{{"namespace": "Microsoft.DocumentDb/databaseAccounts",
			"name": []string{"DataUsage", "DocumentCount", "DocumentQuota"}}}}}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	assert.Nil(t, errs)
	assert.NotEmpty(t, events)
}


func TestData(t *testing.T) {
	config := test.GetConfig(t, "storage")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
