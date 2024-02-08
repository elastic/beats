// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/activedirectory/internal/activedirectory"
)

var (
	usersBucket = []byte("users")
	stateBucket = []byte("state")

	whenChangedKey = []byte("when_changed")
	lastSyncKey    = []byte("last_sync")
	lastUpdateKey  = []byte("last_update")
)

//go:generate stringer -type State
//go:generate go-licenser -license Elastic
type State int

const (
	Discovered State = iota + 1
	Modified
)

type User struct {
	activedirectory.Entry `json:"activedirectory"`
	State                 State `json:"state"`
}

// stateStore wraps a kvstore.Transaction and provides convenience methods for
// accessing and store relevant data within the kvstore database.
type stateStore struct {
	tx *kvstore.Transaction

	// whenChanged is the last whenChanged time in the set of
	// users and their associated groups.
	whenChanged time.Time

	// lastSync and lastUpdate are the times of the first update
	// or sync operation of users/groups.
	lastSync   time.Time
	lastUpdate time.Time
	users      map[string]*User
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
		users: make(map[string]*User),
		tx:    tx,
	}

	err = s.tx.Get(stateBucket, lastSyncKey, &s.lastSync)
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last sync time from state: %w", err)
	}
	err = s.tx.Get(stateBucket, lastUpdateKey, &s.lastUpdate)
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last update time from state: %w", err)
	}
	err = s.tx.Get(stateBucket, whenChangedKey, &s.whenChanged)
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last change time from state: %w", err)
	}

	err = s.tx.ForEach(usersBucket, func(key, value []byte) error {
		var u User
		err = json.Unmarshal(value, &u)
		if err != nil {
			return fmt.Errorf("unable to unmarshal user from state: %w", err)
		}
		s.users[u.ID] = &u

		return nil
	})
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get users from state: %w", err)
	}

	return &s, nil
}

// storeUser stores a user. If the user does not exist in the store, then the
// user will be marked as discovered. Otherwise, the user will be marked
// as modified.
func (s *stateStore) storeUser(u activedirectory.Entry) *User {
	su := User{Entry: u}
	if existing, ok := s.users[u.ID]; ok {
		su.State = Modified
		*existing = su
	} else {
		su.State = Discovered
		s.users[u.ID] = &su
	}
	return &su
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
		if err == nil {
			return
		}
		rollbackErr := s.tx.Rollback()
		if rollbackErr == nil {
			err = fmt.Errorf("multiple errors during statestore close: %w", errors.Join(err, rollbackErr))
		}
	}()

	if !s.lastSync.IsZero() {
		err = s.tx.Set(stateBucket, lastSyncKey, &s.lastSync)
		if err != nil {
			return fmt.Errorf("unable to save last sync time to state: %w", err)
		}
	}
	if !s.lastUpdate.IsZero() {
		err = s.tx.Set(stateBucket, lastUpdateKey, &s.lastUpdate)
		if err != nil {
			return fmt.Errorf("unable to save last update time to state: %w", err)
		}
	}
	if !s.whenChanged.IsZero() {
		err = s.tx.Set(stateBucket, whenChangedKey, &s.whenChanged)
		if err != nil {
			return fmt.Errorf("unable to save last change time to state: %w", err)
		}
	}

	for key, value := range s.users {
		err = s.tx.Set(usersBucket, []byte(key), value)
		if err != nil {
			return fmt.Errorf("unable to save user %q to state: %w", key, err)
		}
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
