// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/registry"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// Expiration duration for the global state.
const defaultExpirationDuration = 3 * time.Minute

// Default path to the Amcache hive.
const defaultHivePath = "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"

// GlobalState is a shared state object that holds cached Amcache entries.
// It is needed because loading and parsing the Amcache hive can be slow,
// so we want to avoid doing it on every query. Instead, we load it once and
// cache the results for a configurable duration.
//
// the amcache hive has multiple tables that can be queried, and when joins
// are performed, osquery may call the generate function for each table multiple
// times per query.  This can lead to significant performance issues if the
// hive is reloaded each time.  By caching the results in a global state object,
// we can avoid this overhead.
//
// The cache is not populated until the first query is made, to avoid
// unnecessary work if the tables are not used.  Additionally, the cache
// is refreshed at query time if it has expired. But will not be updated until
// the next query, even if it is expired.
//
// TODO: Make sure that keeping this data in memory is not a problem for osquerybeat
//
//	in general

type cachedTables map[tables.TableName][]tables.Entry

func newCachedTables() cachedTables {
	cachedTables := cachedTables{}
	for _, amcacheTable := range tables.AllAmcacheTables() {
		cachedTables[amcacheTable.Name] = nil
	}
	return cachedTables
}

type AmcacheState struct {
	cache              cachedTables
	hivePath           string
	expirationDuration time.Duration
	lock               sync.RWMutex
	timer              *time.Timer
}

// Global variables for the gInstance and a mutex to protect it.
var (
	instance *AmcacheState
)

// newAmcacheState creates a nnw AmcacheState instance with the default configuration.
func newAmcacheState(hivePath string, expirationDuration time.Duration) *AmcacheState {
	state := &AmcacheState{hivePath: hivePath, expirationDuration: expirationDuration, cache: nil}
	state.timer = time.AfterFunc(expirationDuration, func() {
		state.clearCache()
	})
	return state
}

// GetAmcacheState returns the singleton AmcacheState instance.
func GetAmcacheState() *AmcacheState {
	if instance == nil {
		instance = newAmcacheState(defaultHivePath, defaultExpirationDuration)
	}
	return instance
}

func (gs *AmcacheState) clearCache() {
	gs.lock.Lock()
	defer gs.lock.Unlock()
	gs.cache = nil
}

// Update reloads the Amcache hive and repopulates all cached data.
func (gs *AmcacheState) updateLockHeld(log *logger.Logger) error {
	// Reload the registry
	log.Infof("Reading the Amcache hive from %s", gs.hivePath)
	regParser, _, err := registry.LoadRegistry(gs.hivePath, log)
	if err != nil {
		return fmt.Errorf("failed to load amcache registry from %s: %w", gs.hivePath, err)
	}

	// Clear the cache before repopulating
	gs.cache = newCachedTables()

	// Repopulate all caches.
	for _, amcacheTable := range tables.AllAmcacheTables() {
		// Get the entries from the registry.
		entries, err := tables.GetEntriesFromRegistry(amcacheTable, regParser, log)
		if err != nil {
			log.Warningf("failed to get entries for table %s: %v", amcacheTable.Name, err)
			gs.cache[amcacheTable.Name] = nil
			continue
		}

		log.Infof("updating cache for table %s with %d entries", amcacheTable.Name, len(entries))
		// Update the cache for this table.
		gs.cache[amcacheTable.Name] = entries
	}
	gs.timer.Reset(gs.expirationDuration)
	return nil
}

// GetCachedEntries returns the cached entries for a given Amcache table and filter list.
// The lock is held for the duration of the function.
// This is to avoid race conditions when the cache is updated while the function is running.
// This function accepts a list of filters to filter the entries by.
// If the list of filters is empty, all entries are returned.
// If the list of filters is not empty, only the entries that match all filters are returned.
// When no filters are provided, the returned slice references the
// internal cache and must not be modified. If you need to modify
// the slice, make a copy first.
// When filters are provided, a new slice is returned.
func (gs *AmcacheState) GetCachedEntries(amcacheTable tables.AmcacheTable, filterList []filters.Filter, log *logger.Logger) ([]tables.Entry, error) {
	gs.lock.Lock()
	defer gs.lock.Unlock()

	if gs.cache == nil {
		log.Infof("updating amcache cache due to expiration")
		err := gs.updateLockHeld(log)
		if err != nil {
			return nil, err
		}
	}

	cachedTableEntries := gs.cache[amcacheTable.Name]

	if len(filterList) == 0 {
		return cachedTableEntries, nil
	}

	var result []tables.Entry
	// Filter the entries by the filters.
	// Filters are evaluated as AND operations.
	for _, entry := range cachedTableEntries {
		matches := true
		for _, filter := range filterList {
			if !filter.Matches(entry) {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, entry)
		}
	}
	return result, nil
}
