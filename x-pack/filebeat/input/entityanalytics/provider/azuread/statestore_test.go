// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	"github.com/elastic/elastic-agent-libs/logp"
)

func testSetupStore(t *testing.T, filename string) *kvstore.Store {
	store, err := kvstore.NewStore(logp.L(), filename, 0644)
	require.NoError(t, err)

	return store
}

func testCleanupStore(store *kvstore.Store, filename string) {
	_ = store.Close()
	_ = os.Remove(filename)
}

func testAssertValueEquals(t *testing.T, store *kvstore.Store, bucket, key, value []byte) {
	t.Helper()

	var gotValue []byte

	err := store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		var err error

		gotValue, err = tx.GetBytes(bucket, key)

		return err
	})
	require.NoError(t, err)
	require.Equal(t, value, gotValue)
}

func testAssertJSONValueEquals(t *testing.T, store *kvstore.Store, bucket, key []byte, value any) {
	t.Helper()

	valueData, err := json.Marshal(&value)
	require.NoError(t, err)

	testAssertValueEquals(t, store, bucket, key, valueData)
}

func testStoreSetJSONValue(t *testing.T, store *kvstore.Store, bucket, key []byte, value any) {
	err := store.RunTransaction(true, func(tx *kvstore.Transaction) error {
		return tx.Set(bucket, key, &value)
	})

	require.NoError(t, err)
}

func TestStateStore_New(t *testing.T) {
	dbFilename := "TestStateStore_New.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	// Inject test values into store.
	lastSync, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	require.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastSyncKey, &lastSync)
	lastUpdate, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:50:04.546457-05:00")
	require.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastUpdateKey, &lastUpdate)
	usersLink := "users-link"
	devicesLink := "devices-link"
	groupsLink := "groups-link"
	testStoreSetJSONValue(t, store, stateBucket, usersLinkKey, &usersLink)
	testStoreSetJSONValue(t, store, stateBucket, devicesLinkKey, &devicesLink)
	testStoreSetJSONValue(t, store, stateBucket, groupsLinkKey, &groupsLink)

	ss, err := newStateStore(store)
	require.NoError(t, err)
	defer ss.close(false)

	require.Equal(t, lastSync, ss.lastSync)
	require.Equal(t, lastUpdate, ss.lastUpdate)
	require.Equal(t, usersLink, ss.usersLink)
	require.Equal(t, devicesLink, ss.devicesLink)
	require.Equal(t, groupsLink, ss.groupsLink)
}

func TestStateStore_Close(t *testing.T) {
	dbFilename := "TestStateStore_Close.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	ss, err := newStateStore(store)
	require.NoError(t, err)

	ss.lastSync, err = time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	require.NoError(t, err)
	ss.lastUpdate, err = time.Parse(time.RFC3339Nano, "2023-01-12T08:50:04.546457-05:00")
	require.NoError(t, err)

	ss.usersLink = "users-link"
	ss.devicesLink = "devices-link"
	ss.groupsLink = "groups-link"

	user1ID := uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87")
	device1ID := uuid.MustParse("adbbe40a-0627-4328-89f1-88cac84dbc7f")
	group1ID := uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80")
	group2ID := uuid.MustParse("ec8b17ae-ce9d-4099-97ee-4a959638bc29")

	ss.users = map[uuid.UUID]*fetcher.User{
		user1ID: {
			ID: user1ID,
			Fields: map[string]interface{}{
				"userPrincipalName": "user.one@example.com",
				"mail":              "user.one@example.com",
				"displayName":       "User One",
				"givenName":         "User",
				"surname":           "One",
				"jobTitle":          "Software Engineer",
				"mobilePhone":       "123-555-1000",
				"businessPhones":    []any{"123-555-0122"},
			},
			MemberOf:           collections.NewUUIDSet(group1ID),
			TransitiveMemberOf: collections.NewUUIDSet(group1ID),
			Modified:           false,
			Deleted:            false,
		},
	}
	ss.devices = map[uuid.UUID]*fetcher.Device{
		device1ID: {
			ID: device1ID,
			Fields: map[string]interface{}{
				"accountEnabled":         true,
				"deviceId":               "2fbbb8f9-ff67-4a21-b867-a344d18a4198",
				"displayName":            "DESKTOP-LETW452G",
				"operatingSystem":        "Windows",
				"operatingSystemVersion": "10.0.19043.1337",
				"physicalIds":            []interface{}{},
				"extensionAttributes": map[string]interface{}{
					"extensionAttribute1": "BYOD-Device",
				},
				"alternativeSecurityIds": []interface{}{
					map[string]interface{}{
						"type":             "2",
						"identityProvider": nil,
						"key":              "DGFSGHSGGTH345A...35DSFH0A",
					},
				},
			},
			MemberOf:           collections.NewUUIDSet(group1ID),
			TransitiveMemberOf: collections.NewUUIDSet(group1ID),
			Modified:           false,
			Deleted:            false,
		},
	}
	ss.groups = map[uuid.UUID]*fetcher.Group{
		group1ID: {
			ID:   group1ID,
			Name: "group1",
			Members: []fetcher.Member{
				{
					ID:   user1ID,
					Type: fetcher.MemberUser,
				},
				{
					ID:   device1ID,
					Type: fetcher.MemberDevice,
				},
			},
		},
	}
	ss.relationships.AddEdge(group1ID, group2ID)

	err = ss.close(true)
	require.NoError(t, err)

	testAssertJSONValueEquals(t, store, stateBucket, lastSyncKey, &ss.lastSync)
	testAssertJSONValueEquals(t, store, stateBucket, lastUpdateKey, &ss.lastUpdate)
	testAssertJSONValueEquals(t, store, stateBucket, usersLinkKey, &ss.usersLink)
	testAssertJSONValueEquals(t, store, stateBucket, devicesLinkKey, &ss.devicesLink)
	testAssertJSONValueEquals(t, store, stateBucket, groupsLinkKey, &ss.groupsLink)

	gotUsers := map[uuid.UUID]*fetcher.User{}
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		err = tx.ForEach(usersBucket, func(key, value []byte) error {
			var u fetcher.User
			err = json.Unmarshal(value, &u)
			require.NoError(t, err)
			gotUsers[u.ID] = &u

			return nil
		})
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)
	require.EqualValues(t, ss.users, gotUsers)

	gotDevices := map[uuid.UUID]*fetcher.Device{}
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		err = tx.ForEach(devicesBucket, func(key, value []byte) error {
			var d fetcher.Device
			err = json.Unmarshal(value, &d)
			require.NoError(t, err)
			gotDevices[d.ID] = &d

			return nil
		})
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)
	require.EqualValues(t, ss.devices, gotDevices)

	gotGroups := map[uuid.UUID]*fetcher.Group{}
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		err = tx.ForEach(groupsBucket, func(key, value []byte) error {
			var g fetcher.Group
			err = json.Unmarshal(value, &g)
			require.NoError(t, err)
			gotGroups[g.ID] = &g

			return nil
		})
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)
	// Workaround for verification, Members is not persisted on groups, so it is
	// nil-ed out here so the assert below works.
	for _, v := range ss.groups {
		v.Members = nil
	}
	require.EqualValues(t, ss.groups, gotGroups)

	testAssertJSONValueEquals(t, store, relationshipsBucket, groupMembershipsKey, &ss.relationships)
}

func TestGetLastSync(t *testing.T) {
	dbFilename := "TestGetLastSync.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	testTime, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	require.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastSyncKey, &testTime)

	got, gotErr := getLastSync(store)

	require.NoError(t, gotErr)
	require.Equal(t, testTime, got)
}

func TestGetLastUpdate(t *testing.T) {
	dbFilename := "TestGetLastUpdate.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	testTime, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	require.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastUpdateKey, &testTime)

	got, gotErr := getLastUpdate(store)

	require.NoError(t, gotErr)
	require.Equal(t, testTime, got)
}

func TestErrIsItemFound(t *testing.T) {
	tests := map[string]struct {
		In   error
		Want bool
	}{
		"bucket-not-found": {
			In:   kvstore.ErrBucketNotFound,
			Want: true,
		},
		"key-not-found": {
			In:   kvstore.ErrKeyNotFound,
			Want: true,
		},
		"invalid error": {
			In:   errors.New("test error"),
			Want: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := errIsItemNotFound(tc.In)

			require.Equal(t, tc.Want, got)
		})
	}
}
