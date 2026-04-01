// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	mockauth "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator/mock"
	mockfetcher "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher/mock"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestAzure_DoFetch(t *testing.T) {
	tests := []struct {
		dataset     string
		wantUsers   bool
		wantDevices bool
	}{
		{dataset: "", wantUsers: true, wantDevices: true},
		{dataset: "all", wantUsers: true, wantDevices: true},
		{dataset: "users", wantUsers: true, wantDevices: false},
		{dataset: "devices", wantUsers: false, wantDevices: true},
	}

	for _, test := range tests {
		t.Run(test.dataset, func(t *testing.T) {
			suffix := test.dataset
			if suffix != "" {
				suffix = "_" + suffix
			}
			dbFilename := fmt.Sprintf("TestAzure_DoFetch%s.db", suffix)
			store := testSetupStore(t, dbFilename)
			t.Cleanup(func() {
				testCleanupStore(store, dbFilename)
			})

			a := azure{
				conf:    conf{Dataset: test.dataset},
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

			if test.wantUsers {
				require.Equal(t, wantModifiedUsers.Values(), gotUsers.Values())
			} else {
				require.Equal(t, 0, gotUsers.Len())
			}
			if test.wantDevices {
				require.Equal(t, wantModifiedDevices.Values(), gotDevices.Values())
			} else {
				require.Equal(t, 0, gotDevices.Len())
			}
		})
	}
}

func TestAzure_DoFetch_MFAEnrichment(t *testing.T) {
	dbFilename := "TestAzure_DoFetch_MFAEnrichment.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	a := azure{
		conf:    conf{Dataset: "users", EnrichWith: []string{"mfa"}},
		logger:  logp.L(),
		auth:    mockauth.New(""),
		fetcher: mockfetcher.New(),
	}

	ss, err := newStateStore(store)
	require.NoError(t, err)
	defer ss.close(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err = a.doFetch(ctx, ss, false)
	require.NoError(t, err)

	// Verify that MFA details were populated for users that have matching
	// entries in MFAResponse.
	for userID, wantMFA := range mockfetcher.MFAResponse {
		u, ok := ss.users[userID]
		require.Truef(t, ok, "expected user %q to be in state", userID)
		require.NotNilf(t, u.MFA, "expected user %q to have MFA details", userID)
		require.Equal(t, wantMFA, u.MFA)
	}
}

func TestAzure_DoFetch_NoMFAEnrichment(t *testing.T) {
	dbFilename := "TestAzure_DoFetch_NoMFAEnrichment.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	// No enrich_with set: MFA field must remain nil.
	a := azure{
		conf:    conf{Dataset: "users"},
		logger:  logp.L(),
		auth:    mockauth.New(""),
		fetcher: mockfetcher.New(),
	}

	ss, err := newStateStore(store)
	require.NoError(t, err)
	defer ss.close(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err = a.doFetch(ctx, ss, false)
	require.NoError(t, err)

	for _, u := range ss.users {
		require.Nil(t, u.MFA, "expected user %q to have no MFA details when enrich_with is not set", u.ID)
	}
}
