// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/kvstore"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/jamf/internal/jamf"
)

var (
	computersBucket = []byte("computers")
	stateBucket     = []byte("state")

	lastSyncKey      = []byte("last_sync")
	lastUpdateKey    = []byte("last_update")
	computersLinkKey = []byte("devices_link")
)

//go:generate stringer -type State
//go:generate go-licenser -license Elastic
type State int

const (
	Discovered State = iota + 1
	Modified
	Deleted
)

type Computer struct {
	jamf.Computer `json:"properties"`
	State         State `json:"state"`
}

// stateStore wraps a kvstore.Transaction and provides convenience methods for
// accessing and store relevant data within the kvstore database.
type stateStore struct {
	tx *kvstore.Transaction

	// lastSync and lastUpdate are the times of the first update
	// or sync operation of users/devices.
	lastSync   time.Time
	lastUpdate time.Time
	computers  map[string]*Computer
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
		computers: make(map[string]*Computer),
		tx:        tx,
	}

	err = s.tx.Get(stateBucket, lastSyncKey, &s.lastSync)
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last sync time from state: %w", err)
	}
	err = s.tx.Get(stateBucket, lastUpdateKey, &s.lastUpdate)
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get last update time from state: %w", err)
	}

	err = s.tx.ForEach(computersBucket, func(key, value []byte) error {
		var c Computer
		err = json.Unmarshal(value, &c)
		if err != nil {
			return fmt.Errorf("unable to unmarshal computer from state: %w", err)
		}
		if c.Udid == nil {
			return fmt.Errorf("did not get computer id from state: %s", value)
		}
		s.computers[*c.Udid] = &c

		return nil
	})
	if err != nil && !errIsItemNotFound(err) {
		return nil, fmt.Errorf("unable to get devices from state: %w", err)
	}

	return &s, nil
}

// storeComputer stores a computer. If the computer does not exist in the store,
// then the computer will be marked as discovered. Otherwise, the user will be
// marked as modified or deleted depending on the state of the IsManaged field.
// changed will be returned true if the record is updated in any way.
func (s *stateStore) storeComputer(c jamf.Computer) (_ *Computer, changed bool) {
	if c.Udid == nil {
		return nil, false
	}
	stored, ok := s.computers[*c.Udid]
	if !ok {
		// Whether this is managed or not, it is discovered. The next sync
		// will change its state to Deleted if it is unmanaged.
		curr := &Computer{Computer: c, State: Discovered}
		s.computers[*c.Udid] = curr
		return curr, true
	}

	changed = !c.Equal(stored.Computer)
	stored.Computer = c
	if c.IsManaged != nil || !*c.IsManaged { // Assume no flag means unmanaged.
		stored.State = Deleted
		return stored, changed
	}
	if changed {
		stored.State = Modified
	}
	return stored, changed
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
		if rollbackErr != nil {
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

	for key, value := range s.computers {
		err = s.tx.Set(computersBucket, []byte(key), value)
		if err != nil {
			return fmt.Errorf("unable to save device %q to state: %w", key, err)
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
