// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package vpc

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestData(t *testing.T) {
	namespaceIs := func(namespace string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue("aws.cloudwatch.namespace")
			return err == nil && v == namespace
		}
	}

	dataFiles := []struct {
		namespace string
		path      string
	}{
		{"AWS/NATGateway", "./_meta/data.json"},
		{"AWS/VPN", "./_meta/data_vpn.json"},
		{"AWS/TransitGateway", "./_meta/data_transit_gateway.json"},
	}

	config, info := mtest.GetConfigForTest("vpc", "300s")
	if info != "" {
		t.Skip("Skipping TestData: " + info)
	}

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("namespace: %s", df.namespace), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, namespaceIs(df.namespace))
		})
	}
}
