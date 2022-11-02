// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
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
			"compute service with nil regions slice in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: nil}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\"",
		},
		{
			"compute service with empty regions in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{}}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\"",
		},
		{
			"compute service with no regions provided in config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\"",
		},
		{
			"compute service with 1 region in regions config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{"us-central1"}}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND resource.labels.zone = starts_with(\"us-central1\")",
		},
		{
			"compute service with 2 regions in regions config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{"us-central1", "europe-west2"}}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND (resource.labels.zone = starts_with(\"us-central1\") OR resource.labels.zone = starts_with(\"europe-west2\"))",
		},
		{
			"compute service with 2 regions in regions config (trim)",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{"us-central1-*", "europe-west2-*"}}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND (resource.labels.zone = starts_with(\"us-central1-\") OR resource.labels.zone = starts_with(\"europe-west2-\"))",
		},
		{
			"compute service with 3 regions in regions config",
			"compute",
			"compute.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{"us-central1", "europe-west2", "europe-north1"}}, logger: logger},
			"metric.type=\"compute.googleapis.com/firewall/dropped_bytes_count\" AND (resource.labels.zone = starts_with(\"us-central1\") OR resource.labels.zone = starts_with(\"europe-west2\") OR resource.labels.zone = starts_with(\"europe-north1\"))",
		},
		{
			"gke service with 2 regions in regions config",
			"gke",
			"gke.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{"us-central1", "europe-west2"}}, logger: logger},
			"metric.type=\"gke.googleapis.com/firewall/dropped_bytes_count\" AND (resource.label.location = starts_with(\"us-central1\") OR resource.label.location = starts_with(\"europe-west2\"))",
		},
		{
			"storage service with region in config",
			"storage",
			"storage.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Region: "us-central1"}, logger: logger},
			"metric.type=\"storage.googleapis.com/firewall/dropped_bytes_count\" AND resource.label.location = starts_with(\"us-central1\")",
		},
		{
			"storage service with 2 regions in regions config",
			"storage",
			"storage.googleapis.com/firewall/dropped_bytes_count",
			metricsRequester{config: config{Regions: []string{"us-central1", "europe-west2"}}, logger: logger},
			"metric.type=\"storage.googleapis.com/firewall/dropped_bytes_count\" AND (resource.label.location = starts_with(\"us-central1\") OR resource.label.location = starts_with(\"europe-west2\"))",
		},
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

func TestBuildRegionsFilter(t *testing.T) {
	r := metricsRequester{}

	cases := []struct {
		title            string
		serviceZoneLabel string
		regions          []string
		expectedFilter   string
	}{
		{
			"nil regions slice",
			gcp.ComputeResourceLabelZone,
			nil,
			"",
		},
		{
			"empty regions slice",
			gcp.ComputeResourceLabelZone,
			[]string{},
			"",
		},
		{
			"default zone label us-central1",
			gcp.DefaultResourceLabelZone,
			[]string{"us-central1"},
			"resource.label.zone = starts_with(\"us-central1\")",
		},
		{
			"compute zone label us-central1",
			gcp.ComputeResourceLabelZone,
			[]string{"us-central1"},
			"resource.labels.zone = starts_with(\"us-central1\")",
		},
		{
			"gke location label us-central1",
			gcp.GKEResourceLabelLocation,
			[]string{"us-central1"},
			"resource.label.location = starts_with(\"us-central1\")",
		},
		{
			"storage location label us-central1",
			gcp.StorageResourceLabelLocation,
			[]string{"us-central1"},
			"resource.label.location = starts_with(\"us-central1\")",
		},
		{
			"compute zone label 2 regions",
			gcp.ComputeResourceLabelZone,
			[]string{"us-central1", "europe-west2"},
			"(resource.labels.zone = starts_with(\"us-central1\") OR resource.labels.zone = starts_with(\"europe-west2\"))",
		},
		{
			"compute zone label 2 regions (trim)",
			gcp.ComputeResourceLabelZone,
			[]string{"us-central1-*", "europe-west2-*"},
			"(resource.labels.zone = starts_with(\"us-central1-\") OR resource.labels.zone = starts_with(\"europe-west2-\"))",
		},
		{
			"compute zone label 3 regions",
			gcp.ComputeResourceLabelZone,
			[]string{"us-central1", "europe-west2", "europe-north1"},
			"(resource.labels.zone = starts_with(\"us-central1\") OR resource.labels.zone = starts_with(\"europe-west2\") OR resource.labels.zone = starts_with(\"europe-north1\"))",
		},
		{
			"gke location label 2 regions",
			gcp.GKEResourceLabelLocation,
			[]string{"us-central1", "europe-west2"},
			"(resource.label.location = starts_with(\"us-central1\") OR resource.label.location = starts_with(\"europe-west2\"))",
		},
		{
			"storage location label 2 regions",
			gcp.StorageResourceLabelLocation,
			[]string{"us-central1", "europe-west2"},
			"(resource.label.location = starts_with(\"us-central1\") OR resource.label.location = starts_with(\"europe-west2\"))",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := r.buildRegionsFilter(c.regions, c.serviceZoneLabel)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}
