// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws
// +build integration,aws

package billing

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "billing", "24h")

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	resultTypeIs := func(resultType string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue("aws.billing.group_definition.key")
			if err == nil {
				// Check for Cost Explorer billing metrics
				k, _ := e.GetValue("aws.billing.group_by." + v.(string))
				exists, _ := aws.StringInSlice(k.(string), []string{"NoAZ", "NoInstanceType"})
				if !exists {
					return v == resultType
				}
			}
			// Check for CloudWatch billing metrics
			exists, err := e.HasKey("aws.billing.EstimatedCharges")
			return err == nil && strconv.FormatBool(exists) == resultType
		}
	}

	dataFiles := []struct {
		resultType string
		path       string
	}{
		{"AZ", "./_meta/data.json"},
		{"INSTANCE_TYPE", "./_meta/data_group_by_instance_type.json"},
		{"LINKED_ACCOUNT", "./_meta/data_group_by_linked_account.json"},
		{"true", "./_meta/data_cloudwatch.json"},
	}

	config := mtest.GetConfigForTest(t, "billing", "24h")
	config = addCostExplorerToConfig(config)
	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		metricSet.WriteEvents(t, "/")
		t.Run(fmt.Sprintf("result type: %s", df.resultType), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, resultTypeIs(df.resultType))
		})
	}
}

func addCostExplorerToConfig(config map[string]interface{}) map[string]interface{} {
	costExplorerConfig := map[string]interface{}{}
	costExplorerConfig["group_by_dimension_keys"] = []string{"AZ", "INSTANCE_TYPE", "LINKED_ACCOUNT"}
	config["cost_explorer_config"] = costExplorerConfig
	return config
}
