// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build aws

package cost

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestData(t *testing.T) {
	resultTypeIs := func(resultType string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue("aws.cost.group_definition.key")
			if v == "aws:createdBy" {
				t, err := e.GetValue("aws.cost.resourceTags.aws:createdBy")
				if t == resultType {
					return err == nil
				}
			}
			return err == nil && v == resultType
		}
	}

	dataFiles := []struct {
		resultType string
		path       string
	}{
		{"", "./_meta/data.json"},
		{"aws:createdBy", "./_meta/data_group_by_tag.json"},
		{"AZ", "./_meta/data_group_by_az.json"},
	}

	config := mtest.GetConfigForTest(t, "cost", "24h")
	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("result type: %s", df.resultType), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, resultTypeIs(df.resultType))
		})
	}
}
