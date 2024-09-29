// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build aws && integration && awshealth

package awshealth

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

// TODO
// There seems to be problem with Flatten() function of type M map[string]interface{}
// The Flatten function returns aws.awshealth.affected_entities instead of aws.awshealth.affected_entities.aws_account_id, needed for nested type.

func TestFetch(t *testing.T) {
	config := mtest.GetConfigForTest(t, "awshealth", "24h")
	metricSet := mbtest.NewReportingMetricSetV2WithContext(t, config)
	events, errs := mbtest.ReportingFetchV2WithContext(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	mbtest.TestMetricsetFieldsDocumented(t, metricSet, events)
}

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "awshealth", "1h")
	config["dataset_id"] = "master_aws_awshealth"

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
