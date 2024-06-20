// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
)

var (
	usersBucket         = []byte("users")
	groupsBucket        = []byte("groups")
	devicesBucket       = []byte("devices")
	relationshipsBucket = []byte("relationships")
	stateBucket         = []byte("state")

	lastSyncKey         = []byte("last_sync")
	lastUpdateKey       = []byte("last_update")
	usersLinkKey        = []byte("users_link")
	devicesLinkKey      = []byte("devices_link")
	groupsLinkKey       = []byte("groups_link")
	groupMembershipsKey = []byte("group_memberships")
)

// stateStore wraps a kvstore.Transaction and provides convenience methods for
// accessing and store relevant data within the kvstore database.
type stateStore struct {
	tx *kvstore.Transaction

	lastSync      time.Time
	lastUpdate    time.Time
	usersLink     string
	devicesLink   string
	groupsLink    string
	users         map[uuid.UUID]*fetcher.User
	devices       map[uuid.UUID]*fetcher.Device
	groups        map[uuid.UUID]*fetcher.Group
	relationships collections.UUIDTree
}

// newStateStore creates a new instance of stateStore. It will open a new write
// transaction on the kvstore and load values from the database. Since this
// opens a write transaction, only one instance of stateStore may be created
// at a time. The close function must be called to release the transaction lock
// on the kvstore database.
func newStateStore(store *kvstore.Store) (*stateStore, error) {
	tx, err := store.BeginTx(true)
	if err != nil {
		return nil, fmt.Errorf("unable to open state store transaction: %w", err)
	}

	s := stateStore{
		users:   map[uuid.UUID]*fetcher.User{},
		devices: map[uuid.UUID]*fetcher.Device{},
		groups:  map[uuid.UUID]*fetcher.Group{},
		tx:      tx,
	}

	if err = s.tx.Get(stateBucket, lastSyncKey, &s.lastSync); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last sync time from state: %w", err)
	}
	if err = s.tx.Get(stateBucket, lastUpdateKey, &s.lastUpdate); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last update time from state: %w", err)
	}
	if err = s.tx.Get(stateBucket, usersLinkKey, &s.usersLink); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get users link from state: %w", err)
	}
	if err = s.tx.Get(stateBucket, devicesLinkKey, &s.devicesLink); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get devices link from state: %w", err)
	}
	if err = s.tx.Get(stateBucket, groupsLinkKey, &s.groupsLink); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get groups link from state: %w", err)
	}

	if err = s.tx.ForEach(usersBucket, func(key, value []byte) error {
		var u fetcher.User
		if err = json.Unmarshal(value, &u); err != nil {
			return fmt.Errorf("unable to unmarshal user from state: %w", err)
		}
		s.users[u.ID] = &u

		return nil
	}); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get users from state: %w", err)
	}

	if err = s.tx.ForEach(devicesBucket, func(key, value []byte) error {
		var d fetcher.Device
		if err = json.Unmarshal(value, &d); err != nil {
			return fmt.Errorf("unable to unmarshal device from state: %w", err)
		}
		s.devices[d.ID] = &d

		return nil
	}); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get devices from state: %w", err)
	}

	if err = s.tx.ForEach(groupsBucket, func(key, value []byte) error {
		var g fetcher.Group
		if err = json.Unmarshal(value, &g); err != nil {
			return fmt.Errorf("unable to unmarshal group from state: %w", err)
		}
		s.groups[g.ID] = &g

		return nil
	}); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get users from state: %w", err)
	}

	if err = s.tx.Get(relationshipsBucket, groupMembershipsKey, &s.relationships); err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get groups relationships from state: %w", err)
	}

	return &s, nil
}

// storeUser stores a user. If the user does not exist in the store, then the
// user will be marked as discovered. Otherwise, the user will be marked
// as modified.
func (s *stateStore) storeUser(u *fetcher.User) {
	if existing, ok := s.users[u.ID]; ok {
		u.Modified = true
		existing.Merge(u)
	} else if !u.Deleted {
		u.Discovered = true
		s.users[u.ID] = u
	}
}

// storeDevice stores a device. If the device does not exist in the store, then the
// device will be marked as discovered. Otherwise, the device will be marked
// as modified.
func (s *stateStore) storeDevice(d *fetcher.Device) {
	if existing, ok := s.devices[d.ID]; ok {
		d.Modified = true
		existing.Merge(d)
	} else if !d.Deleted {
		d.Discovered = true
		s.devices[d.ID] = d
	}
}

// storeGroup stores a group. If the group already exists, it will be overwritten.
func (s *stateStore) storeGroup(g *fetcher.Group) {
	s.groups[g.ID] = g
}

// close will close out the stateStore. If commit is true, the staged values on the
// stateStore will be set in the kvstore database, and the transaction will be
// committed. Otherwise, all changes will be discarded and the transaction will
// be rolled back. The stateStore must NOT be used after close is called, rather,
// a new stateStore should be created.
func (s *stateStore) close(commit bool) (err error) {
	if !commit {
		return s.tx.Rollback()
	}

	// Fallback in case one of the statements below fails. If everything is
	// successful and Commit is called, then this call to Rollback will be a no-op.
	defer func() {
		rollbackErr := s.tx.Rollback()
		if rollbackErr == nil {
			return
		}

		if err != nil {
			err = fmt.Errorf("multiple errors during statestore close: %w", errors.Join(err, rollbackErr))
		} else {
			err = rollbackErr
		}
	}()

	if !s.lastSync.IsZero() {
		if err = s.tx.Set(stateBucket, lastSyncKey, &s.lastSync); err != nil {
			return fmt.Errorf("unable to save last sync time to state: %w", err)
		}
	}
	if !s.lastUpdate.IsZero() {
		if err = s.tx.Set(stateBucket, lastUpdateKey, &s.lastUpdate); err != nil {
			return fmt.Errorf("unable to save last update time to state: %w", err)
		}
	}
	if s.usersLink != "" {
		if err = s.tx.Set(stateBucket, usersLinkKey, &s.usersLink); err != nil {
			return fmt.Errorf("unable to save users link to state: %w", err)
		}
	}
	if s.devicesLink != "" {
		if err = s.tx.Set(stateBucket, devicesLinkKey, &s.devicesLink); err != nil {
			return fmt.Errorf("unable to save devices link to state: %w", err)
		}
	}
	if s.groupsLink != "" {
		if err = s.tx.Set(stateBucket, groupsLinkKey, &s.groupsLink); err != nil {
			return fmt.Errorf("unable to save groups link to state: %w", err)
		}
	}

	for key, value := range s.users {
		if err = s.tx.Set(usersBucket, key[:], value); err != nil {
			return fmt.Errorf("unable to save user %q to state: %w", key, err)
		}
	}
	for key, value := range s.devices {
		if err = s.tx.Set(devicesBucket, key[:], value); err != nil {
			return fmt.Errorf("unable to save device %q to state: %w", key, err)
		}
	}
	for key, value := range s.groups {
		if err = s.tx.Set(groupsBucket, key[:], value); err != nil {
			return fmt.Errorf("unable to save group %q to state: %w", key, err)
		}
	}

	if err = s.tx.Set(relationshipsBucket, groupMembershipsKey, &s.relationships); err != nil {
		return fmt.Errorf("unable to save group memberships to state: %w", err)
	}

	return s.tx.Commit()
}

// getLastSync retrieves the last full synchronization time from the kvstore
// database. If the value doesn't exist, a zero time.Time is returned.
func getLastSync(store *kvstore.Store) (time.Time, error) {
	var t time.Time
	err := store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		return tx.Get(stateBucket, lastSyncKey, &t)
	})

	return t, err
}

// getLastUpdate retrieves the last incremental update time from the kvstore
// database. If the value doesn't exist, a zero time.Time is returned.
func getLastUpdate(store *kvstore.Store) (time.Time, error) {
	var t time.Time
	err := store.RunTransaction(false, func(tx *kvstore.Transaction) error {
		return tx.Get(stateBucket, lastUpdateKey, &t)
	})

	return t, err
}

// errIsItemNotFound returns true if the error represents an item not found
// error (bucket not found or key not found).
func errIsItemNotFound(err error) bool {
	return errors.Is(err, kvstore.ErrBucketNotFound) || errors.Is(err, kvstore.ErrKeyNotFound)
}
