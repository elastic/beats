// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestStringInSlice(t *testing.T) {
	cases := []struct {
		title          string
		m              string
		region         string
		zone           string
		expectedFilter string
	}{
		{
			"construct filter with zone",
			"compute.googleapis.com/instance/cpu/utilization",
			"",
			"us-east1-b",
			"metric.type=\"compute.googleapis.com/instance/cpu/utilization\" AND resource.labels.zone = \"us-east1-b\"",
		},
		{
			"construct filter with region",
			"compute.googleapis.com/instance/cpu/utilization",
			"us-east1",
			"",
			"metric.type=\"compute.googleapis.com/instance/cpu/utilization\" AND resource.labels.zone = starts_with(\"us-east1\")",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := constructFilter(c.m, c.region, c.zone)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}

func TestGetFilterForMetric(t *testing.T) {
	var logger = logp.NewLogger("test")
	cases := []struct {
		title          string
		m              string
		r              stackdriverMetricsRequester
		expectedFilter string
	}{
		{
			"compute service with zone in config",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			stackdriverMetricsRequester{config: config{Zone: "us-central1-a"}},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = \"us-central1-a\"",
		},
		{
			"pubsub service with zone in config",
			"pubsub.googleapis.com/subscription/ack_message_count",
			stackdriverMetricsRequester{config: config{Zone: "us-central1-a"}},
			"metric.type=\"pubsub.googleapis.com/subscription/ack_message_count\"",
		},
		{
			"loadbalancing service with zone in config",
			"loadbalancing.googleapis.com/https/backend_latencies",
			stackdriverMetricsRequester{config: config{Zone: "us-central1-a"}},
			"metric.type=\"loadbalancing.googleapis.com/https/backend_latencies\"",
		},
		{
			"compute service with region in config",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			stackdriverMetricsRequester{config: config{Region: "us-east1"}},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = starts_with(\"us-east1\")",
		},
		{
			"pubsub service with region in config",
			"pubsub.googleapis.com/subscription/ack_message_count",
			stackdriverMetricsRequester{config: config{Region: "us-east1"}},
			"metric.type=\"pubsub.googleapis.com/subscription/ack_message_count\"",
		},
		{
			"loadbalancing service with region in config",
			"loadbalancing.googleapis.com/https/backend_latencies",
			stackdriverMetricsRequester{config: config{Region: "us-east1"}},
			"metric.type=\"loadbalancing.googleapis.com/https/backend_latencies\"",
		},
		{
			"compute service with both region and zone in config",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			stackdriverMetricsRequester{config: config{Region: "us-central1", Zone: "us-central1-a"}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = starts_with(\"us-central1\")",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := c.r.getFilterForMetric(c.m)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}
