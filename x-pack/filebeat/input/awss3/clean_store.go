// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/go-concert/unison"
)

type commitWriteState struct {
	time.Time
}

// cleanStore runs periodically at interval checking for states to purge from the store
func cleanStore(canceler unison.Canceler, logger *logp.Logger, store *statestore.Store, states *States, interval time.Duration) {
	started := time.Now()
	timed.Periodic(canceler, interval, func() error {
		gcStore(logger, started, store, states)
		return nil
	})
}

// gcStore looks for states to remove and deletes these. `gcStore` receives
// the start timestamp of the cleaner as reference.
func gcStore(logger *logp.Logger, started time.Time, store *statestore.Store, states *States) {
	logger.Debugf("Start store cleanup")
	defer logger.Debugf("Done store cleanup")

	keys := gcFind(states, started, time.Now())
	if len(keys) == 0 {
		logger.Debugf("No entries to remove were found")
		return
	}

	if err := gcClean(store, states, keys); err != nil {
		logger.Errorf("Failed to remove all entries from the registry: %+v", err)
	}

	if err := store.Set(awsS3WriteCommitStateKey, commitWriteState{time.Now()}); err != nil {
		logger.Errorf("Failed to write commit time to the registry: %+v", err)
	}
}

// gcFind searches the store of states that can be removed. A set of keys to delete is returned.
// if the state is marked as stored it will be purged. If the state is not marked as stored
// it will be purged if state.LastModified or the time of when the cleaner started
// (whichever is the latest) is in the past.
func gcFind(states *States, started, now time.Time) map[string]struct{} {
	keys := map[string]struct{}{}
	for _, state := range states.GetStates() {
		reference := state.LastModified
		if !state.Stored && started.After(reference) {
			reference = started
		}

		if reference.Before(now) && !state.Stored {
			keys[state.Id] = struct{}{}
			continue
		}

		// it is stored, forget
		if state.Stored {
			keys[state.Id] = struct{}{}
		}
	}

	return keys
}

// gcClean removes key value pairs in the removeSet from the store.
// If deletion in the persistent store fails the entry is kept in memory and
// eventually cleaned up later.
func gcClean(store *statestore.Store, states *States, removeSet map[string]struct{}) error {
	for key := range removeSet {
		if err := store.Remove(awsS3ObjectStatePrefix + key); err != nil {
			return err
		}

		states.Delete(key)
	}
	return nil
}
