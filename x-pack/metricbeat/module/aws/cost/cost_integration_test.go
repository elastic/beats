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
	resultTypeIs := func(resultTypeIsEmpty bool) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue("aws.cost.resourceTags.aws:createdBy")
			return err == nil && (v == "") == resultTypeIsEmpty
		}
	}

	dataFiles := []struct {
		resultTypeIsEmpty bool
		path              string
	}{
		{true, "./_meta/data.json"},
		{false, "./_meta/data_group_by.json"},
	}

	config := mtest.GetConfigForTest(t, "cost", "24h")
	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("result type: %t", df.resultTypeIsEmpty), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, resultTypeIs(df.resultTypeIsEmpty))
		})
	}
}
