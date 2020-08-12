// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package billing

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestData(t *testing.T) {
	resultTypeIs := func(resultType string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue("aws.billing.group_definition.key")
			// Check for cost explorer metrics with group by TAG
			if v == "aws:createdBy" {
				exists, err := e.HasKey("aws.billing.resourceTags.aws:createdBy")
				if err != nil {
					// aggregated total value when there is no `aws:createdBy` tag key
					return strconv.FormatBool(exists) == resultType
				}
				// when tag value exists for `aws:createdBy`
				tag, err := e.GetValue("aws.billing.resourceTags.aws:createdBy")
				return strconv.FormatBool(tag != "") == resultType
			}

			if err == nil {
				// group by AZ
				return v == resultType
			} else {
				// Check for CloudWatch billing metrics
				exists, err := e.HasKey("aws.billing.EstimatedCharges")
				return err == nil && strconv.FormatBool(exists) == resultType
			}
		}
	}

	dataFiles := []struct {
		resultType string
		path       string
	}{
		{"false", "./_meta/data.json"},
		{"true", "./_meta/data_group_by_tag.json"},
		{"AZ", "./_meta/data_group_by_az.json"},
		{"true", "./_meta/data_cloudwatch.json"},
	}

	config := mtest.GetConfigForTest(t, "billing", "24h")
	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("result type: %s", df.resultType), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, resultTypeIs(df.resultType))
		})
	}
}
