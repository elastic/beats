// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration
// +build cloudfoundry

package container

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry/mtest"
)

func TestFetch(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("cloudfoundry"))

	t.Run("v1", func(t *testing.T) {
		testFetch(t, "v1")
	})

	t.Run("v2", func(t *testing.T) {
		testFetch(t, "v2")
	})
}

func testFetch(t *testing.T, version string) {
	config := mtest.GetConfig(t, "container")
	config["version"] = version

	ms := mbtest.NewPushMetricSetV2(t, config)
	events := mbtest.RunPushMetricSetV2(60*time.Second, 1, ms)

	require.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	config := mtest.GetConfig(t, "container")

	ms := mbtest.NewPushMetricSetV2(t, config)
	events := mbtest.RunPushMetricSetV2(60*time.Second, 1, ms)

	require.NotEmpty(t, events)

	beatEvent := mbtest.StandardizeEvent(ms, events[0])
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}
