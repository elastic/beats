// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws
// +build integration,aws

package elb

import (
	"fmt"
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestData(t *testing.T) {
	namespaceIs := func(namespace string) func(e mapstr.M) bool {
		return func(e mapstr.M) bool {
			v, err := e.GetValue("aws.cloudwatch.namespace")
			return err == nil && v == namespace
		}
	}

	dataFiles := []struct {
		namespace string
		path      string
	}{
		{"AWS/ELB", "./_meta/data.json"},
		{"AWS/ApplicationELB", "./_meta/data_alb.json"},
		{"AWS/NetworkELB", "./_meta/data_nlb.json"},
	}

	config := mtest.GetConfigForTest(t, "elb", "300s")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("namespace: %s", df.namespace), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, namespaceIs(df.namespace))
		})
	}
}
