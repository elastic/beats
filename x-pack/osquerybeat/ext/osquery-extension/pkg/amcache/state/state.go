// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"log"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
)

// Expiration duration for the global state.
const defaultExpirationDuration = 3 * time.Minute

// Default path to the Amcache hive.
const defaultHivePath = "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"

// Config holds configuration for the GlobalState.
type Config struct {
	HivePath           string
	ExpirationDuration time.Duration
}

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

type CachedTables map[tables.TableType]map[string][]tables.Entry

func NewCachedTables() CachedTables {
	cachedTables := make(CachedTables)
	for _, tableType := range tables.AllTableTypes() {
		cachedTables[tableType] = make(map[string][]tables.Entry)
	}
	return cachedTables
}

type AmcacheGlobalState struct {
	Cache       CachedTables
	Config      *Config
	Lock        sync.RWMutex
	LastUpdated time.Time
}

// Global variables for the gInstance and a mutex to protect it.
var (
	gInstance *AmcacheGlobalState = &AmcacheGlobalState{
		Config: &Config{HivePath: defaultHivePath, ExpirationDuration: defaultExpirationDuration},
		Cache:  NewCachedTables(),
	}
)

// GetAmcacheGlobalState is the public accessor for the singleton.
// It checks for expiration and re-creates the instance if needed.
func GetAmcacheGlobalState() *AmcacheGlobalState {
	return gInstance
}

// Update reloads the Amcache hive and repopulates all cached data.
func (gs *AmcacheGlobalState) Update() {
	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	// Reload the registry
	registry, err := tables.LoadRegistry(gs.Config.HivePath)
	if err != nil {
		log.Printf("error opening amcache registry: %v", err)
		return
	}

	// Repopulate all caches.
	for _, tableType := range tables.AllTableTypes() {
		// keyPath represents each relevant key in the Amcache hive such as "Root\InventoryApplication"
		keyPath := tableType.GetHiveKey()

		// Initialize the map for this keyPath
		gs.Cache[tableType] = make(map[string][]tables.Entry)

		// Get entries for this keyPath from the loaded registry
		entries, err := tables.GetEntriesFromRegistry(tableType, registry)
		if err != nil {
			// Log the error for this key and continue so we don't leave a nil map.
			log.Printf("error getting %s entries: %v", keyPath, err)
		}
		if entries == nil {
			entries = make(map[string][]tables.Entry)
		}
		gs.Cache[tableType] = entries
	}
	gs.LastUpdated = time.Now()
}

// GetMarshalledEntriesForKeyPath retrieves marshalled entries for a given key path and optional entry IDs.
// it returns a slice of maps, where each map represents an entry/row in the table.
// keyPath is the Amcache key path such as "Root\InventoryApplication".
// ids are optional entry IDs to filter the results. If no IDs are provided, all entries for the keyPath are returned.
// each amcache key has a field that can be used as an ID to filter on, for example ProgramId for Application entries.
func (gs *AmcacheGlobalState) GetCachedEntries(tableType tables.TableType, ids ...string) []tables.Entry {
	gs.UpdateIfNeeded()

	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	result := make([]tables.Entry, 0)
	cachedTableEntries := gs.Cache[tableType]

	// If no IDs are provided, return all entries for the keyPath
	if len(ids) == 0 {
		for _, byId := range cachedTableEntries {
			for _, entry := range byId {
				result = append(result, entry)
			}
		}
		return result
	}

	for _, entryId := range ids {
		if entries, ok := cachedTableEntries[entryId]; ok {
			for _, entry := range entries {
				result = append(result, entry)
			}
		}
	}
	return result
}

// UpdateIfNeeded checks if the cache has expired and updates it if necessary.
func (gs *AmcacheGlobalState) UpdateIfNeeded() {
	gs.Lock.RLock()
	lastUpdated := gs.LastUpdated
	expirationDuration := gs.Config.ExpirationDuration
	gs.Lock.RUnlock()

	if time.Since(lastUpdated) > expirationDuration {
		gs.Update()
	}
}
