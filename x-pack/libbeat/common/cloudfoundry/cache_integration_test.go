// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && cloudfoundry && !aix
// +build integration,cloudfoundry,!aix

package cloudfoundry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	cftest "github.com/elastic/beats/v8/x-pack/libbeat/common/cloudfoundry/test"
)

func TestGetApps(t *testing.T) {
	var conf Config
	err := common.MustNewConfigFrom(cftest.GetConfigFromEnv(t)).Unpack(&conf)
	require.NoError(t, err)

	log := logp.NewLogger("cloudfoundry")
	hub := NewHub(&conf, "filebeat", log)

	client, err := hub.Client()
	require.NoError(t, err)
	apps, err := client.ListApps()
	require.NoError(t, err)

	t.Logf("%d applications available", len(apps))

	t.Run("request one of the available applications", func(t *testing.T) {
		if len(apps) == 0 {
			t.Skip("no apps in account?")
		}
		client, err := hub.ClientWithCache()
		require.NoError(t, err)
		defer client.Close()

		guid := apps[0].Guid
		app, err := client.GetAppByGuid(guid)
		assert.Equal(t, guid, app.Guid)
		assert.NoError(t, err)
	})

	t.Run("handle error when application is not available", func(t *testing.T) {
		client, err := hub.ClientWithCache()
		require.NoError(t, err)
		defer client.Close()

		testNotExists := func(t *testing.T) {
			app, err := client.GetAppByGuid("notexists")
			assert.Nil(t, app)
			assert.Error(t, err)
			assert.True(t, cfclient.IsAppNotFoundError(err), "Error found: %v", err)
		}

		var firstTimeDuration time.Duration
		t.Run("first call", func(t *testing.T) {
			startTime := time.Now()
			testNotExists(t)
			firstTimeDuration = time.Now().Sub(startTime)
		})

		t.Run("second call, in cache, faster, same response", func(t *testing.T) {
			for i := 0; i < 10; i++ {
				startTime := time.Now()
				testNotExists(t)
				require.True(t, firstTimeDuration > time.Now().Sub(startTime))
			}
		})
	})
}
