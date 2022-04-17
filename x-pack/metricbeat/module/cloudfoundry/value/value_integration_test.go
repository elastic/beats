// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && cloudfoundry && !aix
// +build integration,cloudfoundry,!aix

package value

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/logp"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/cloudfoundry/mtest"
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
	config := mtest.GetConfig(t, "value")
	config["version"] = version

	ms := mbtest.NewPushMetricSetV2(t, config)
	events := mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)

	require.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	config := mtest.GetConfig(t, "value")

	ms := mbtest.NewPushMetricSetV2(t, config)
	events := mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)

	require.NotEmpty(t, events)

	beatEvent := mbtest.StandardizeEvent(ms, events[0])
	mtest.CleanFields(beatEvent)
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}
