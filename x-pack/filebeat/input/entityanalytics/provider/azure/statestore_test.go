// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azure/fetcher"
	"github.com/elastic/elastic-agent-libs/logp"
)

func testSetupStore(filename string) *kvstore.Store {
	store, err := kvstore.NewStore(logp.L(), filename, 0644)
	if err != nil {
		panic(err)
	}

	return store
}

func testCleanupStore(store *kvstore.Store, filename string) {
	_ = store.Close()
	_ = os.Remove(filename)
}

func testAssertValueEquals(t *testing.T, store *kvstore.Store, bucket, key, value []byte) {
	var gotValue []byte

	err := store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		var err error

		gotValue, err = tx.GetBytes(bucket, key)

		return err
	})
	assert.NoError(t, err)
	assert.Equal(t, value, gotValue)
}

func testAssertJSONValueEquals(t *testing.T, store *kvstore.Store, bucket, key []byte, value any) {
	valueData, err := json.Marshal(&value)
	assert.NoError(t, err)

	testAssertValueEquals(t, store, bucket, key, valueData)
}

func testStoreSetJSONValue(t *testing.T, store *kvstore.Store, bucket, key []byte, value any) {
	err := store.RunTransaction(true, func(tx *kvstore.Transaction) error {
		return tx.Set(bucket, key, &value)
	})

	assert.NoError(t, err)
}

func TestStateStore_New(t *testing.T) {
	dbFilename := "TestStateStore_New.db"
	store := testSetupStore(dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	// Inject test values into store.
	lastSync, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	assert.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastSyncKey, &lastSync)
	lastUpdate, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:50:04.546457-05:00")
	assert.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastUpdateKey, &lastUpdate)
	usersLink := "users-link"
	groupsLink := "groups-link"
	testStoreSetJSONValue(t, store, stateBucket, usersLinkKey, &usersLink)
	testStoreSetJSONValue(t, store, stateBucket, groupsLinkKey, &groupsLink)

	ss, err := newStateStore(store)
	assert.NoError(t, err)
	defer ss.close(false)

	assert.Equal(t, lastSync, ss.lastSync)
	assert.Equal(t, lastUpdate, ss.lastUpdate)
	assert.Equal(t, usersLink, ss.usersLink)
	assert.Equal(t, groupsLink, ss.groupsLink)
}

func TestStateStore_Close(t *testing.T) {
	dbFilename := "TestStateStore_Close.db"
	store := testSetupStore(dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	ss, err := newStateStore(store)
	assert.NoError(t, err)

	ss.lastSync, err = time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	assert.NoError(t, err)
	ss.lastUpdate, err = time.Parse(time.RFC3339Nano, "2023-01-12T08:50:04.546457-05:00")
	assert.NoError(t, err)

	ss.usersLink = "users-link"
	ss.groupsLink = "groups-link"

	user1ID := uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87")
	group1ID := uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80")

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
			MemberOf:           collections.NewSet[uuid.UUID](group1ID),
			TransitiveMemberOf: collections.NewSet[uuid.UUID](group1ID),
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
			},
		},
	}
	ss.relationships = collections.NewTree[uuid.UUID]()
	ss.relationships.AddVertex(group1ID)

	err = ss.close(true)
	assert.NoError(t, err)

	testAssertJSONValueEquals(t, store, stateBucket, lastSyncKey, &ss.lastSync)
	testAssertJSONValueEquals(t, store, stateBucket, lastUpdateKey, &ss.lastUpdate)
	testAssertJSONValueEquals(t, store, stateBucket, usersLinkKey, &ss.usersLink)
	testAssertJSONValueEquals(t, store, stateBucket, groupsLinkKey, &ss.groupsLink)

	gotUsers := map[uuid.UUID]*fetcher.User{}
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		err = tx.ForEach(usersBucket, func(key, value []byte) error {
			var u fetcher.User
			err = json.Unmarshal(value, &u)
			assert.NoError(t, err)
			gotUsers[u.ID] = &u

			return nil
		})
		assert.NoError(t, err)

		return nil
	})
	assert.NoError(t, err)
	assert.EqualValues(t, ss.users, gotUsers)

	gotGroups := map[uuid.UUID]*fetcher.Group{}
	err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		err = tx.ForEach(groupsBucket, func(key, value []byte) error {
			var g fetcher.Group
			err = json.Unmarshal(value, &g)
			assert.NoError(t, err)
			gotGroups[g.ID] = &g

			return nil
		})
		assert.NoError(t, err)

		return nil
	})
	assert.NoError(t, err)
	// Workaround for verification, Members is not persisted on groups, so it is
	// nil-ed out here so the assert below works.
	for _, v := range ss.groups {
		v.Members = nil
	}
	assert.EqualValues(t, ss.groups, gotGroups)

	testAssertJSONValueEquals(t, store, relationshipsBucket, groupMembershipsKey, ss.relationships)
}

func TestGetLastSync(t *testing.T) {
	dbFilename := "TestGetLastSync.db"
	store := testSetupStore(dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	testTime, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	assert.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastSyncKey, &testTime)

	got, gotErr := getLastSync(store)

	assert.NoError(t, gotErr)
	assert.Equal(t, testTime, got)
}

func TestGetLastUpdate(t *testing.T) {
	dbFilename := "TestGetLastUpdate.db"
	store := testSetupStore(dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	testTime, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	assert.NoError(t, err)
	testStoreSetJSONValue(t, store, stateBucket, lastUpdateKey, &testTime)

	got, gotErr := getLastUpdate(store)

	assert.NoError(t, gotErr)
	assert.Equal(t, testTime, got)
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

			assert.Equal(t, tc.Want, got)
		})
	}
}
