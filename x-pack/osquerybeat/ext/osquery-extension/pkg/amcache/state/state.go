// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"log"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application_file"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/application_shortcut"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/driver_binary"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables/device_pnp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/interfaces"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/utilities"
	"sync"
	"time"
)

// Expiration duration for the singleton instance.
const expirationDuration = 3 * time.Minute

// GlobalState holds our shared state and its creation time.
type GlobalState struct {
	application     map[string][]interfaces.Entry
	applicationFile map[string][]interfaces.Entry
	applicationShortcut map[string][]interfaces.Entry
	driverBinary    map[string][]interfaces.Entry
	devicePnp       map[string][]interfaces.Entry
	lock            sync.RWMutex
	creationTime    time.Time
}

// Global variables for the instance and a mutex to protect it.
var (
	instance   *GlobalState = &GlobalState{}
	instanceMu sync.Mutex
	hivePath   string = "C:\\Windows\\AppCompat\\Programs\\Amcache.hve"
)

// GetInstance is the public accessor for the singleton.
// It checks for expiration and re-creates the instance if needed.
func GetInstance() *GlobalState {
	return instance
}

func (gs *GlobalState) ResetLockHeld() {
	registry, err := utilities.LoadRegistry(hivePath)
	if err != nil {
		log.Printf("error opening amcache registry: %v", err)
		return
	}
	instance.application, err = application.GetApplicationEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting application entries: %v", err)
		return
	}
	instance.applicationFile, err = application_file.GetApplicationFileEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting application file entries: %v", err)
		return
	}
	instance.applicationShortcut, err = application_shortcut.GetApplicationShortcutEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting application shortcut entries: %v", err)
		return
	}
	instance.driverBinary, err = driver_binary.GetDriverBinaryEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting driver binary entries: %v", err)
		return
	}
	instance.devicePnp, err = device_pnp.GetDevicePnpEntriesFromRegistry(registry)
	if err != nil {
		log.Printf("error getting device pnp entries: %v", err)
		return
	}
	instance.creationTime = time.Now()
}

func (gs *GlobalState) ResetIfNeeded() {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	if gs.IsExpired() {
		log.Println("GlobalState needs reset, reloading hive from:", hivePath)
		gs.ResetLockHeld()
	}
}

func SetHivePath(path string) {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	log.Println("Setting Amcache hive path to:", hivePath)
	hivePath = path
	instance.ResetLockHeld()
}

func (gs *GlobalState) IsExpired() bool {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return time.Since(gs.creationTime) > expirationDuration
}

func (gs *GlobalState) GetApplicationEntries(programId ...string) []interfaces.Entry {
	gs.ResetIfNeeded()

	gs.lock.Lock()
	defer gs.lock.Unlock()

	var entries []interfaces.Entry
	if len(programId) == 0 {
		for _, entry := range gs.application {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range programId {
			if entry, ok := gs.application[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetApplicationFileEntries(programId ...string) []interfaces.Entry {
	gs.ResetIfNeeded()

	gs.lock.Lock()
	defer gs.lock.Unlock()

	var entries []interfaces.Entry
	if len(programId) == 0 {
		for _, entry := range gs.applicationFile {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range programId {
			if entry, ok := gs.applicationFile[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetApplicationShortcutEntries(programId ...string) []interfaces.Entry {
	gs.ResetIfNeeded()
	gs.lock.Lock()
	defer gs.lock.Unlock()
	var entries []interfaces.Entry
	if len(programId) == 0 {
		for _, entry := range gs.applicationShortcut {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range programId {
			if entry, ok := gs.applicationShortcut[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetDriverBinaryEntries(driverId ...string) []interfaces.Entry {
	gs.ResetIfNeeded()
	gs.lock.Lock()
	defer gs.lock.Unlock()
	var entries []interfaces.Entry
	if len(driverId) == 0 {
		for _, entry := range gs.driverBinary {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range driverId {
			if entry, ok := gs.driverBinary[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

func (gs *GlobalState) GetDevicePnpEntries(deviceId ...string) []interfaces.Entry {
	gs.ResetIfNeeded()
	gs.lock.Lock()
	defer gs.lock.Unlock()
	var entries []interfaces.Entry
	if len(deviceId) == 0 {
		for _, entry := range gs.devicePnp {
			entries = append(entries, entry...)
		}
	} else {
		for _, id := range deviceId {
			if entry, ok := gs.devicePnp[id]; ok {
				entries = append(entries, entry...)
			}
		}
	}
	return entries
}

