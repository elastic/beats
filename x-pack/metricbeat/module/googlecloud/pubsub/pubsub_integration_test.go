// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build googlecloud

package pubsub

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud/metrics"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	metricPrefixIs := func(metricPrefix string) func(e common.MapStr) bool {
		return func(e common.MapStr) bool {
			v, err := e.GetValue(metricPrefix)
			return err == nil && v != nil
		}
	}

	dataFiles := []struct {
		metricPrefix string
		path         string
	}{
		{"googlecloud.pubsub", "./_meta/data.json"},
		{"googlecloud.pubsub.snapshot", "./_meta/data_snapshot.json"},
		{"googlecloud.pubsub.subscription", "./_meta/data_subscription.json"},
		{"googlecloud.pubsub.topic", "./_meta/data_topic.json"},
	}

	config := metrics.GetConfigForTest(t, "pubsub")

	for _, df := range dataFiles {
		metricSet := mbtest.NewFetcher(t, config)
		t.Run(fmt.Sprintf("metric prefix: %s", df.metricPrefix), func(t *testing.T) {
			metricSet.WriteEventsCond(t, df.path, metricPrefixIs(df.metricPrefix))
		})
	}
}
