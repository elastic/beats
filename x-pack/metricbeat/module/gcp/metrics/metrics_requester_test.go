// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

func TestGetFilterForMetric(t *testing.T) {
	var logger = logp.NewLogger("test")
	cases := []struct {
		title          string
		s              string
		m              string
		r              metricsRequester
		expectedFilter string
	}{
		{
			"compute service with zone in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Zone: "us-central1-a"}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = starts_with(\"us-central1-a\")",
		},
		{
			"pubsub service with zone in config",
			"pubsub",
			"pubsub.googleapis.com/subscription/ack_message_count",
			metricsRequester{config: config{Zone: "us-central1-a"}, logger: logger},
			"metric.type=\"pubsub.googleapis.com/subscription/ack_message_count\"",
		},
		{
			"loadbalancing service with zone in config",
			"loadbalancing",
			"loadbalancing.googleapis.com/https/backend_latencies",
			metricsRequester{config: config{Zone: "us-central1-a"}, logger: logger},
			"metric.type=\"loadbalancing.googleapis.com/https/backend_latencies\"",
		},
		{
			"compute service with region in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Region: "us-east1"}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = starts_with(\"us-east1\")",
		},
		{
			"pubsub service with region in config",
			"pubsub",
			"pubsub.googleapis.com/subscription/ack_message_count",
			metricsRequester{config: config{Region: "us-east1"}, logger: logger},
			"metric.type=\"pubsub.googleapis.com/subscription/ack_message_count\"",
		},
		{
			"loadbalancing service with region in config",
			"loadbalancing",
			"loadbalancing.googleapis.com/https/backend_latencies",
			metricsRequester{config: config{Region: "us-east1"}, logger: logger},
			"metric.type=\"loadbalancing.googleapis.com/https/backend_latencies\"",
		},
		{
			"compute service with both region and zone in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Region: "us-central1", Zone: "us-central1-a"}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = starts_with(\"us-central1\")",
		},
		{
			"compute uptime with partial region",
			"compute",
			"compute.googleapis.com/instance/uptime",
			metricsRequester{config: config{Region: "us-west"}, logger: logger},
			"metric.type=\"compute.googleapis.com/instance/uptime\" AND resource.labels.zone = starts_with(\"us-west\")",
		},
		{
			"compute uptime with partial zone",
			"compute",
			"compute.googleapis.com/instance/uptime",
			metricsRequester{config: config{Zone: "us-west1-"}, logger: logger},
			"metric.type=\"compute.googleapis.com/instance/uptime\" AND resource.labels.zone = starts_with(\"us-west1-\")",
		},
		{
			"compute uptime with wildcard in region",
			"compute",
			"compute.googleapis.com/instance/uptime",
			metricsRequester{config: config{Region: "us-*"}, logger: logger},
			"metric.type=\"compute.googleapis.com/instance/uptime\" AND resource.labels.zone = starts_with(\"us-\")",
		},
		{
			"compute uptime with wildcard in zone",
			"compute",
			"compute.googleapis.com/instance/uptime",
			metricsRequester{config: config{Zone: "us-west1-*"}, logger: logger},
			"metric.type=\"compute.googleapis.com/instance/uptime\" AND resource.labels.zone = starts_with(\"us-west1-\")",
		},
		{
			"compute service with no region/zone in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\"",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := c.r.getFilterForMetric(c.s, c.m)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}

func TestGetTimeIntervalAligner(t *testing.T) {
	cases := []struct {
		title            string
		ingestDelay      time.Duration
		samplePeriod     time.Duration
		collectionPeriod *duration.Duration
		inputAligner     string
		expectedAligner  string
	}{
		{
			"test collectionPeriod equals to samplePeriod",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&duration.Duration{
				Seconds: int64(60),
			},
			"",
			"ALIGN_NONE",
		},
		{
			"test collectionPeriod larger than samplePeriod",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&duration.Duration{
				Seconds: int64(300),
			},
			"ALIGN_MEAN",
			"ALIGN_MEAN",
		},
		{
			"test collectionPeriod smaller than samplePeriod",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&duration.Duration{
				Seconds: int64(30),
			},
			"ALIGN_MAX",
			"ALIGN_NONE",
		},
		{
			"test collectionPeriod equals to samplePeriod with given aligner",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&duration.Duration{
				Seconds: int64(60),
			},
			"ALIGN_MEAN",
			"ALIGN_NONE",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			_, aligner := getTimeIntervalAligner(c.ingestDelay, c.samplePeriod, c.collectionPeriod, c.inputAligner)
			assert.Equal(t, c.expectedAligner, aligner)
		})
	}
}
