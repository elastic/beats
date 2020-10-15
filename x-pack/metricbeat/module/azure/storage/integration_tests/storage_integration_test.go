// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package integration_tests

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pulumi/pulumi/pkg/v2/testing/integration"
	"github.com/pulumi/pulumi/sdk/v2/go/common/apitype"
	"os"
	"path"
	"time"

	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/test"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/storage"
)

const location = "WestEurope"

func GetConfigCredentials(t *testing.T) map[string]string {
	t.Helper()
	clientId, ok := os.LookupEnv("AZURE_CLIENT_ID")
	if !ok {
		t.Fatal("Could not find var AZURE_CLIENT_ID")
	}
	tenantId, ok := os.LookupEnv("AZURE_TENANT_ID")
	if !ok {
		t.Fatal("Could not find var AZURE_TENANT_ID")
	}
	subId, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		t.Fatal("Could not find var AZURE_SUBSCRIPTION_ID")
	}
	return map[string]string{
		"cloud:provider":       "azure",
		"azure:environment":    "public",
		"azure:location":       location,
		"azure:subscriptionId": subId,
		"azure:clientId":       clientId,
		"azure:tenantId":       tenantId,
	}
}

func GetConfigSecret(t *testing.T) map[string]string {
	t.Helper()
	clientSecret, ok := os.LookupEnv("AZURE_CLIENT_SECRET")
	if !ok {
		t.Fatal("Could not find var AZURE_CLIENT_SECRET")
	}
	return map[string]string{
		"azure:clientSecret": clientSecret,
	}
}

func TestFetchMetricset(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Error thrown: " + err.Error())
	}
	dir := path.Join(cwd, "config")
	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Quick:       true,
		SkipRefresh: true,
		Dir:         dir,
		Config:      GetConfigCredentials(t),
		Secrets:     GetConfigSecret(t),
		ExtraRuntimeValidation: func(t *testing.T, stack integration.RuntimeValidationStackInfo) {
			var storageAccount apitype.ResourceV3
			for _, res := range stack.Deployment.Resources {
				if res.Type == "azure:storage/account:Account" {
					storageAccount = res
				}
			}
			assert.NotNil(t, storageAccount)
			// will need some time to gather relevand metric values, else no vlaues are returned
			time.Sleep(150 * time.Second)
			config := test.GetConfig(t, "storage")
			config["resources"] = []map[string]interface{}{{
				"resource_id": storageAccount.ID}}
			metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
			events, errs := mbtest.ReportingFetchV2Error(metricSet)
			assert.Nil(t, errs)
			assert.NotEmpty(t, events)
			timegrain, err := events[0].ModuleFields.GetValue("timegrain")
			if err != nil {
				t.Fatal("Error thrown: " + err.Error())
			}
			assert.Equal(t, timegrain, "PT5M")
			resource, err := events[0].ModuleFields.GetValue("resource")
			if err != nil {
				t.Fatal("Error thrown: " + err.Error())
			}
			res := resource.(common.MapStr)
			assert.Equal(t, res["type"], "Microsoft.Storage/storageAccounts")
			assert.Equal(t, res["id"], storageAccount.ID)
		},
	})
}

func TestData(t *testing.T) {
	config := test.GetConfig(t, "storage")
	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
