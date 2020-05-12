// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tests

import (
	"os"
	"path"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/test"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/pulumi/pulumi/pkg/v2/testing/integration"
)

const (
	resourceGroupName = "observability-beats-test"
	location          = "WestEurope"
	storageAccount    = "storage-account"
)

func TestExamples(t *testing.T) {
	cwd, _ := os.Getwd()
	dir:= path.Join(cwd, "info")
	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Quick:       true,
		SkipRefresh: true,
		Dir:         dir,
		Config: map[string]string{
			"cloud:provider":             "azure",
			"azure:environment":          "public",
			"cloud-azure:location":       location,
			"cloud-azure:subscriptionId": "",
			"cloud-azure:clientId":       "",
			"cloud-azure:tenantId":       "",
		},
		Secrets: map[string]string{
			"cloud-azure:clientSecret": "",
		},
		ExtraRuntimeValidation: func(t *testing.T, stack integration.RuntimeValidationStackInfo) {
			assert.EqualValues(t, stack.Outputs["endpoint"], func(body string) bool {
				return assert.Contains(t, body, "Greetings from Azure App Service!")
			})
		},
	})
}



func TestFetchMetricset(t *testing.T) {
	config := test.GetConfig(t, "storage")
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	config := test.GetConfig(t, "storage")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
