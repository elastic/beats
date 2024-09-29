// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/internal/okta"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestStateStore(t *testing.T) {
	logp.TestingSetup()

	lastSync, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:47:23.296794-05:00")
	if err != nil {
		t.Fatalf("failed to parse lastSync")
	}
	lastUpdate, err := time.Parse(time.RFC3339Nano, "2023-01-12T08:50:04.546457-05:00")
	if err != nil {
		t.Fatalf("failed to parse lastUpdate")
	}
	const (
		usersLink   = "users-link"
		devicesLink = "devices-link"
	)

	t.Run("new", func(t *testing.T) {
		dbFilename := "TestStateStore_New.db"
		store := testSetupStore(t, dbFilename)
		t.Cleanup(func() {
			testCleanupStore(store, dbFilename)
		})

		// Inject test values into store.
		data := []struct {
			key []byte
			val any
		}{
			{key: lastSyncKey, val: lastSync},
			{key: lastUpdateKey, val: lastUpdate},
			{key: usersLinkKey, val: usersLink},
			{key: devicesLinkKey, val: devicesLink},
		}
		for _, kv := range data {
			err := store.RunTransaction(true, func(tx *kvstore.Transaction) error {
				return tx.Set(stateBucket, kv.key, kv.val)
			})
			if err != nil {
				t.Fatalf("failed to set %s: %v", kv.key, err)
			}
		}

		ss, err := newStateStore(store)
		if err != nil {
			t.Fatalf("failed to make new store: %v", err)
		}
		defer ss.close(false)

		checks := []struct {
			name      string
			got, want any
		}{
			{name: "lastSync", got: ss.lastSync, want: lastSync},
			{name: "lastUpdate", got: ss.lastUpdate, want: lastUpdate},
			{name: "usersLink", got: ss.nextUsers, want: usersLink},
			{name: "devicesLink", got: ss.nextDevices, want: devicesLink},
		}
		for _, c := range checks {
			if !cmp.Equal(c.got, c.want) {
				t.Errorf("unexpected results for %s: got:%#v want:%#v", c.name, c.got, c.want)
			}
		}
	})

	t.Run("close", func(t *testing.T) {
		dbFilename := "TestStateStore_Close.db"
		store := testSetupStore(t, dbFilename)
		t.Cleanup(func() {
			testCleanupStore(store, dbFilename)
		})

		wantUsers := map[string]*User{
			"userid": {
				State: Discovered,
				User: okta.User{
					ID:              "userid",
					Status:          "STATUS",
					Created:         time.Now(),
					StatusChanged:   ptr(time.Now()),
					LastLogin:       ptr(time.Now()),
					LastUpdated:     time.Now(),
					PasswordChanged: ptr(time.Now()),
					Type: map[string]interface{}{
						"id": "typeid",
					},
					Profile: map[string]interface{}{
						"login":     "name.surname@example.com",
						"email":     "name.surname@example.com",
						"firstName": "name",
						"lastName":  "surname",
					},
					Credentials: &okta.Credentials{
						Password:         &struct{}{}, // Had a password: not retained.
						RecoveryQuestion: &struct{}{}, // Had a question: not retained.
						Provider: okta.Provider{
							Type: "OKTA",
							Name: ptr("OKTA"),
						},
					},
					Links: okta.HAL{
						"self": map[string]interface{}{
							"href": "https://localhost/api/v1/users/userid",
						},
					},
				},
			},
		}
		wantDevices := map[string]*Device{
			"deviceid": {
				State: Discovered,
				Device: okta.Device{
					ID:          "deviceid",
					Status:      "STATUS",
					Created:     time.Now(),
					LastUpdated: time.Now(),
					Links: okta.HAL{
						"self": map[string]interface{}{
							"href": "https://localhost/api/v1/devices/deviceid",
						},
					},
				},
			},
		}

		ss, err := newStateStore(store)
		if err != nil {
			t.Fatalf("failed to make new store: %v", err)
		}
		ss.lastSync = lastSync
		ss.lastUpdate = lastUpdate
		ss.nextUsers = usersLink
		ss.nextDevices = devicesLink
		ss.users = wantUsers
		ss.devices = wantDevices

		err = ss.close(true)
		if err != nil {
			t.Fatalf("unexpected error closing: %v", err)
		}

		roundTripChecks := []struct {
			name string
			key  []byte
			val  any
		}{
			{name: "lastSyncKey", key: lastSyncKey, val: &ss.lastSync},
			{name: "lastUpdateKey", key: lastUpdateKey, val: &ss.lastUpdate},
			{name: "usersLinkKey", key: usersLinkKey, val: &ss.nextUsers},
			{name: "devicesLinkKey", key: devicesLinkKey, val: &ss.nextDevices},
		}
		for _, check := range roundTripChecks {
			want, err := json.Marshal(check.val)
			if err != nil {
				t.Errorf("unexpected error marshaling %s: %v", check.name, err)
			}
			var got []byte
			err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
				got, err = tx.GetBytes(stateBucket, check.key)
				return err
			})
			if err != nil {
				t.Errorf("unexpected error from store run transaction %s: %v", check.name, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("unexpected result after store round-trip for %s: got:%s want:%s", check.name, got, want)
			}
		}

		users := map[string]*User{}
		err = store.RunTransaction(false, func(tx *kvstore.Transaction) error {
			return tx.ForEach(usersBucket, func(key, value []byte) error {
				var u User
				err = json.Unmarshal(value, &u)
				if err != nil {
					return err
				}
				users[u.ID] = &u
				return nil
			})
		})
		if err != nil {
			t.Errorf("unexpected error from store run transaction: %v", err)
		}
		if !cmp.Equal(wantUsers, users) {
			t.Errorf("unexpected result:\n- want\n+ got\n%s", cmp.Diff(wantUsers, users))
		}
	})

	t.Run("get_last_sync", func(t *testing.T) {
		dbFilename := "TestGetLastSync.db"
		store := testSetupStore(t, dbFilename)
		t.Cleanup(func() {
			testCleanupStore(store, dbFilename)
		})

		err := store.RunTransaction(true, func(tx *kvstore.Transaction) error {
			return tx.Set(stateBucket, lastSyncKey, lastSync)
		})
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		got, err := getLastSync(store)
		if err != nil {
			t.Errorf("unexpected error from getLastSync: %v", err)
		}
		if !lastSync.Equal(got) {
			t.Errorf("unexpected result from getLastSync: got:%v want:%v", got, lastSync)
		}
	})

	t.Run("get_last_update", func(t *testing.T) {
		dbFilename := "TestGetLastUpdate.db"
		store := testSetupStore(t, dbFilename)
		t.Cleanup(func() {
			testCleanupStore(store, dbFilename)
		})

		err := store.RunTransaction(true, func(tx *kvstore.Transaction) error {
			return tx.Set(stateBucket, lastUpdateKey, lastUpdate)
		})
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		got, err := getLastUpdate(store)
		if err != nil {
			t.Errorf("unexpected error from getLastUpdate: %v", err)
		}
		if !lastUpdate.Equal(got) {
			t.Errorf("unexpected result from getLastUpdate: got:%v want:%v", got, lastUpdate)
		}
	})
}

func TestErrIsItemFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "bucket-not-found",
			err:  kvstore.ErrBucketNotFound,
			want: true,
		},
		{
			name: "key-not-found",
			err:  kvstore.ErrKeyNotFound,
			want: true,
		},
		{
			name: "invalid error",
			err:  errors.New("test error"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := errIsItemNotFound(test.err)
			if got != test.want {
				t.Errorf("unexpected result for %s: got:%t want:%t", test.name, got, test.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func testSetupStore(t *testing.T, path string) *kvstore.Store {
	t.Helper()

	store, err := kvstore.NewStore(logp.L(), path, 0644)
	if err != nil {
		t.Fatalf("unexpected error making store: %v", err)
	}
	return store
}

func testCleanupStore(store *kvstore.Store, path string) {
	_ = store.Close()
	_ = os.Remove(path)
}
