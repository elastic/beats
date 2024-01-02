// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestGetTimeIntervalAligner(t *testing.T) {
	cases := []struct {
		title            string
		ingestDelay      time.Duration
		samplePeriod     time.Duration
		collectionPeriod *durationpb.Duration
		inputAligner     string
		expectedAligner  string
	}{
		{
			"test collectionPeriod equals to samplePeriod",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&durationpb.Duration{
				Seconds: int64(60),
			},
			"",
			"ALIGN_NONE",
		},
		{
			"test collectionPeriod larger than samplePeriod",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&durationpb.Duration{
				Seconds: int64(300),
			},
			"ALIGN_MEAN",
			"ALIGN_MEAN",
		},
		{
			"test collectionPeriod smaller than samplePeriod",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&durationpb.Duration{
				Seconds: int64(30),
			},
			"ALIGN_MAX",
			"ALIGN_NONE",
		},
		{
			"test collectionPeriod equals to samplePeriod with given aligner",
			time.Duration(240) * time.Second,
			time.Duration(60) * time.Second,
			&durationpb.Duration{
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

func TestGetFilterForMetric(t *testing.T) {
	logp.DevelopmentSetup(logp.ToObserverOutput())
	var logger = logp.NewLogger("TestGetFilterForMetric")

	cases := []struct {
		title          string
		s              string
		m              string
		r              metricsRequester
		expectedFilter string
	}{
		{
			"compute service with empty config",
			gcp.ServiceCompute,
			"",
			metricsRequester{config: config{}, logger: logger},
			"metric.type=\"dummy\"",
		},
		{
			"compute service with configured region",
			gcp.ServiceCompute,
			"",
			metricsRequester{config: config{Region: "foo"}, logger: logger},
			"metric.type=\"dummy\" AND resource.labels.zone = starts_with(\"foo\")",
		},
		{
			"compute service with configured zone",
			gcp.ServiceCompute,
			"",
			metricsRequester{config: config{Zone: "foo"}, logger: logger},
			"metric.type=\"dummy\" AND resource.labels.zone = starts_with(\"foo\")",
		},
		{
			"compute service with configured regions",
			gcp.ServiceCompute,
			"",
			metricsRequester{config: config{Regions: []string{"foo", "bar"}}, logger: logger},
			"metric.type=\"dummy\" AND (resource.labels.zone = starts_with(\"foo\") OR resource.labels.zone = starts_with(\"bar\"))",
		},
		{
			"compute service with configured region and zone",
			gcp.ServiceCompute,
			"",
			metricsRequester{config: config{Region: "foo", Zone: "bar"}, logger: logger},
			"metric.type=\"dummy\" AND resource.labels.zone = starts_with(\"foo\")",
		},
		{
			"compute service with configured region and regions",
			gcp.ServiceCompute,
			"",
			metricsRequester{config: config{Region: "foobar", Regions: []string{"foo", "bar"}}, logger: logger},
			"metric.type=\"dummy\" AND resource.labels.zone = starts_with(\"foobar\")",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := c.r.getFilterForMetric(c.s, "dummy")
			assert.Equal(t, c.expectedFilter, filter)

			// NOTE: test that we output a log message with the filter value, as this is **extremely**
			// useful to debug issues and we want to make sure is being done.
			logs := logp.ObserverLogs().
				FilterLevelExact(zapcore.DebugLevel). // we are OK it being logged at debug level
				FilterMessageSnippet(filter).
				Len()
			assert.Equal(t, logs, 1)
			// NOTE: cleanup observed logs at each iteration to start with no messages.
			_ = logp.ObserverLogs().TakeAll()

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
			"label",
			nil,
			"",
		},
		{
			"empty regions slice",
			"label",
			[]string{},
			"",
		},
		{
			"with single region",
			"label",
			[]string{"foobar"},
			"label = starts_with(\"foobar\")",
		},
		{
			"with multiple regions",
			"label",
			[]string{"foobar", "foo"},
			"(label = starts_with(\"foobar\") OR label = starts_with(\"foo\"))",
		},
		{
			"with single region (trim)",
			"label",
			[]string{"foobar*"},
			"label = starts_with(\"foobar\")",
		},
		{
			"with multiple regions (trim)",
			"label",
			[]string{"foobar*", "foo*"},
			"(label = starts_with(\"foobar\") OR label = starts_with(\"foo\"))",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			filter := r.buildRegionsFilter(c.regions, c.serviceZoneLabel)
			assert.Equal(t, c.expectedFilter, filter)
		})
	}
}

func TestIsAGlobalService(t *testing.T) {
	cases := []struct {
		title   string
		service string
		global  bool
	}{
		{"empty service name", "", false},
		{"unknown service name", "unknown", false},
		{"CloudFunctions service", gcp.ServiceCloudFunctions, true},
		{"Compute service", gcp.ServiceCompute, false},
		{"GKE service", gcp.ServiceGKE, false},
		{"LoadBalancing service", gcp.ServiceLoadBalancing, true},
		{"PubSub service", gcp.ServicePubsub, true},
		{"Storage service", gcp.ServiceStorage, false},
		{"Firestore service", gcp.ServiceFirestore, true},
		{"Dataproc service", gcp.ServiceDataproc, false},
		{"CloudSQL service", gcp.ServiceCloudSQL, false},
		{"Redis service", gcp.ServiceRedis, false},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			value := isAGlobalService(c.service)
			assert.Equal(t, c.global, value)
		})
	}
}

func TestGetServiceLabelFor(t *testing.T) {
	cases := []struct {
		title    string
		service  string
		expected string
	}{
		{"empty service name", "", gcp.DefaultResourceLabel},
		{"unknown service name", "unknown", gcp.DefaultResourceLabel},
		{"CloudFunctions service", gcp.ServiceCloudFunctions, gcp.DefaultResourceLabel},
		{"Compute service", gcp.ServiceCompute, "resource.labels.zone"},
		{"GKE service", gcp.ServiceGKE, "resource.label.location"},
		{"LoadBalancing service", gcp.ServiceLoadBalancing, gcp.DefaultResourceLabel},
		{"PubSub service", gcp.ServicePubsub, gcp.DefaultResourceLabel},
		{"Storage service", gcp.ServiceStorage, "resource.label.location"},
		{"Firestore service", gcp.ServiceFirestore, gcp.DefaultResourceLabel},
		{"Dataproc service", gcp.ServiceDataproc, "resource.label.region"},
		{"CloudSQL service", gcp.ServiceCloudSQL, "resource.labels.region"},
		{"Redis service", gcp.ServiceRedis, "resource.label.region"},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			value := getServiceLabelFor(c.service)
			assert.Equal(t, c.expected, value)
		})
	}
}

func TestTrimWildcard(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		expected string
	}{
		{"empty", "", ""},
		{"no wildcard", "us-central1", "us-central1"},
		{"with wildcard", "us-central1*", "us-central1"},
		{"with wildcard and special char", "us-central1-*", "us-central1-"},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			value := trimWildcard(c.value)
			assert.Equal(t, c.expected, value)
		})
	}
}

func TestBuildLocationFilter(t *testing.T) {
	logger := logp.NewLogger("TestBuildLocationFilter")
	cases := []struct {
		title     string
		requester metricsRequester
		expected  string
	}{
		{
			"empty config and label service",
			metricsRequester{config: config{}, logger: logger},
			"",
		},
		{
			"region configured and label service",
			metricsRequester{config: config{Region: "foobar"}, logger: logger},
			" AND label = starts_with(\"foobar\")",
		},
		{
			"zone configured and label service",
			metricsRequester{config: config{Zone: "foobar"}, logger: logger},
			" AND label = starts_with(\"foobar\")",
		},
		{
			"regions configured (single) and label service",
			metricsRequester{config: config{Regions: []string{"foobar"}}, logger: logger},
			" AND label = starts_with(\"foobar\")",
		},
		{
			"regions configured (multiple) and label service",
			metricsRequester{config: config{Regions: []string{"foo", "bar"}}, logger: logger},
			" AND (label = starts_with(\"foo\") OR label = starts_with(\"bar\"))",
		},

		{
			"region and zone configured, label service",
			metricsRequester{config: config{Region: "foobar", Zone: "foo"}, logger: logger},
			// NOTE: region takes precedence
			" AND label = starts_with(\"foobar\")",
		},
		{
			"region and regions configured, label service",
			metricsRequester{config: config{Region: "foobar", Regions: []string{"foo", "bar"}}, logger: logger},
			// NOTE: region takes precedence
			" AND label = starts_with(\"foobar\")",
		},
		{
			"zone and regions configured, label service",
			metricsRequester{config: config{Zone: "foobar", Regions: []string{"foo", "bar"}}, logger: logger},
			// NOTE: zone takes precedence
			" AND label = starts_with(\"foobar\")",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			value := c.requester.buildLocationFilter("label", "")
			assert.Equal(t, c.expected, value)
		})
	}
}
