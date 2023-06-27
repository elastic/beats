// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	mockauth "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator/mock"
	mockfetcher "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher/mock"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestAzure_DoFetch(t *testing.T) {
	dbFilename := "TestAzure_DoFetch.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	a := azure{
		logger:  logp.L(),
		auth:    mockauth.New(""),
		fetcher: mockfetcher.New(),
	}

	ss, err := newStateStore(store)
	require.NoError(t, err)
	defer ss.close(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gotUsers, gotDevices, err := a.doFetch(ctx, ss, false)
	require.NoError(t, err)

	var wantModifiedUsers collections.UUIDSet
	for _, v := range mockfetcher.UserResponse {
		wantModifiedUsers.Add(v.ID)
	}
	var wantModifiedDevices collections.UUIDSet
	for _, v := range mockfetcher.DeviceResponse {
		wantModifiedDevices.Add(v.ID)
	}

	require.Equal(t, wantModifiedUsers.Values(), gotUsers.Values())
	require.Equal(t, wantModifiedDevices.Values(), gotDevices.Values())
}
