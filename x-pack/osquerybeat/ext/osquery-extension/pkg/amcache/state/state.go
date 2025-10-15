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
//       in general
//
type GlobalState struct {
	Config              *Config
	Application         map[string][]tables.Entry
	ApplicationFile     map[string][]tables.Entry
	ApplicationShortcut map[string][]tables.Entry
	DriverBinary        map[string][]tables.Entry
	DevicePnp           map[string][]tables.Entry
	Lock                sync.RWMutex
	LastUpdated         time.Time
}

// Global variables for the gInstance and a mutex to protect it.
var (
	gInstance *GlobalState = &GlobalState{Config: &Config{HivePath: defaultHivePath, ExpirationDuration: defaultExpirationDuration}}
)

// GetGlobalState is the public accessor for the singleton.
// It checks for expiration and re-creates the instance if needed.
func GetGlobalState() *GlobalState {
	return gInstance
}

func (gs *GlobalState) Update() {
	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	// Reload the registry and repopulate all caches.
	registry, err := tables.LoadRegistry(gs.Config.HivePath)
	if err != nil {
		log.Printf("error opening amcache registry: %v", err)
		return
	}
	gs.Application, err = tables.GetApplicationEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting application entries: %v", err)
		return
	}
	gs.ApplicationFile, err = tables.GetApplicationFileEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting application file entries: %v", err)
		return
	}
	gs.ApplicationShortcut, err = tables.GetApplicationShortcutEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting application shortcut entries: %v", err)
		return
	}
	gs.DriverBinary, err = tables.GetDriverBinaryEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting driver binary entries: %v", err)
		return
	}
	gs.DevicePnp, err = tables.GetDevicePnpEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting device pnp entries: %v", err)
		return
	}
	gs.LastUpdated = time.Now()
}

func (gs *GlobalState) UpdateIfNeeded() {
	gs.Lock.RLock()
	lastUpdated := gs.LastUpdated
	expirationDuration := gs.Config.ExpirationDuration
	gs.Lock.RUnlock()

	if time.Since(lastUpdated) > expirationDuration {
		gs.Update()
	}
}

func (gs *GlobalState) GetApplicationEntries(programId ...string) []tables.Entry {
	gs.UpdateIfNeeded()

	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	var entries []tables.Entry
	if len(programId) == 0 {
		for _, entry := range gs.Application {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range programId {
			if entry, ok := gs.Application[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetApplicationFileEntries(programId ...string) []tables.Entry {
	gs.UpdateIfNeeded()

	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	var entries []tables.Entry
	if len(programId) == 0 {
		for _, entry := range gs.ApplicationFile {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range programId {
			if entry, ok := gs.ApplicationFile[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetApplicationShortcutEntries(programId ...string) []tables.Entry {
	gs.UpdateIfNeeded()

	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	var entries []tables.Entry
	if len(programId) == 0 {
		for _, entry := range gs.ApplicationShortcut {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range programId {
			if entry, ok := gs.ApplicationShortcut[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetDriverBinaryEntries(driverId ...string) []tables.Entry {
	gs.UpdateIfNeeded()

	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	var entries []tables.Entry
	if len(driverId) == 0 {
		for _, entry := range gs.DriverBinary {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range driverId {
			if entry, ok := gs.DriverBinary[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetDevicePnpEntries(deviceId ...string) []tables.Entry {
	gs.UpdateIfNeeded()

	gs.Lock.Lock()
	defer gs.Lock.Unlock()

	var entries []tables.Entry
	if len(deviceId) == 0 {
		for _, entry := range gs.DevicePnp {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range deviceId {
			if entry, ok := gs.DevicePnp[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}
