// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
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

type CachedTables map[tables.TableName][]tables.Entry

type AmcacheGlobalState struct {
	Cache              CachedTables
	HivePath           string
	ExpirationDuration time.Duration
	Lock               sync.RWMutex
	LastUpdated        time.Time
}

// NewAmcacheGlobalState creates a new AmcacheGlobalState instance with the default configuration.
func newAmcacheGlobalState(hivePath string, expirationDuration time.Duration) *AmcacheGlobalState {
	cachedTables := make(CachedTables)
	for _, amcacheTable := range tables.AllAmcacheTables() {
		cachedTables[amcacheTable.Name] = make([]tables.Entry, 0)
	}
	return &AmcacheGlobalState{HivePath: hivePath, ExpirationDuration: expirationDuration, Cache: cachedTables}
}

// Global variables for the gInstance and a mutex to protect it.
var (
	gInstance *AmcacheGlobalState = newAmcacheGlobalState(defaultHivePath, defaultExpirationDuration)
)

// GetAmcacheGlobalState is the public accessor for the singleton.
// It checks for expiration and re-creates the instance if needed.
func GetAmcacheGlobalState() *AmcacheGlobalState {
	return gInstance
}

// Update reloads the Amcache hive and repopulates all cached data.
func (gs *AmcacheGlobalState) Update(log *logger.Logger) error {
	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	// Reload the registry
	log.Infof("Reading the Amcache hive from %s", gs.HivePath)
	regParser, _, err := registry.LoadRegistry(gs.HivePath, log)
	if err != nil {
		log.Errorf("error opening amcache registry: %v", err)
		return err
	}

	// Repopulate all caches.
	for _, amcacheTable := range tables.AllAmcacheTables() {
		// Initialize the map for this keyPath
		gs.Cache[amcacheTable.Name] = make([]tables.Entry, 0)

		// Get entries for this keyPath from the loaded registry
		entries, err := tables.GetEntriesFromRegistry(amcacheTable, regParser)
		if err != nil {
			// Log the error for this key and continue so we don't leave a nil entry in the map.
			log.Errorf("error getting %s entries: %v", amcacheTable.Name, err)
			continue
		}
		log.Infof("Updating cache for table %s with %d entries", amcacheTable.Name, len(entries))
		gs.Cache[amcacheTable.Name] = append(gs.Cache[amcacheTable.Name], entries...)
	}
	gs.LastUpdated = time.Now()
	return nil
}

// GetCachedEntries returns the cached entries for a given Amcache table and filter list.
func (gs *AmcacheGlobalState) GetCachedEntries(amcacheTable tables.AmcacheTable, filterList []filters.Filter, log *logger.Logger) ([]tables.Entry, error) {
	if gs.IsExpired() {
		log.Infof("Amcache cache is %s old, updating", time.Since(gs.LastUpdated))
		err := gs.Update(log)
		if err != nil {
			log.Errorf("error updating amcache cache: %v", err)
			return nil, err
		}
	}

	gs.Lock.RLock()
	defer gs.Lock.RUnlock()

	result := make([]tables.Entry, 0)
	cachedTableEntries := gs.Cache[amcacheTable.Name]

	if len(filterList) == 0 {
		result = append(result, cachedTableEntries...)
		return result, nil
	}

	for _, entry := range cachedTableEntries {
		for _, filter := range filterList {
			if filter.Matches(entry) {
				result = append(result, entry)
			}
		}
	}
	return result, nil
}

// IsExpired checks if the cache has expired.
func (gs *AmcacheGlobalState) IsExpired() bool {
	gs.Lock.RLock()
	defer gs.Lock.RUnlock()
	return time.Since(gs.LastUpdated) > gs.ExpirationDuration
}