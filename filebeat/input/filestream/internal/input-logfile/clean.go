// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package input_logfile

import (
	"time"

	"github.com/menderesk/go-concert/timed"
	"github.com/menderesk/go-concert/unison"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

// cleaner removes finished entries from the registry file.
type cleaner struct {
	log *logp.Logger
}

// run starts a loop that tries to clean entries from the registry.
// The cleaner locks the store, such that no new states can be created
// during the cleanup phase. Only resources that are finished and whos TTL
// (clean_timeout setting) has expired will be removed.
//
// Resources are considered "Finished" if they do not have a current owner (active input), and
// if they have no pending updates that still need to be written to the registry file after associated
// events have been ACKed by the outputs.
// The event acquisition timestamp is used as reference to clean resources. If a resources was blocked
// for a long time, and the life time has been exhausted, then the resource will be removed immediately
// once the last event has been ACKed.
func (c *cleaner) run(canceler unison.Canceler, store *store, interval time.Duration) {
	started := time.Now()
	timed.Periodic(canceler, interval, func() error {
		gcStore(c.log, started, store)
		return nil
	})
}

// gcStore looks for resources to remove and deletes these. `gcStore` receives
// the start timestamp of the cleaner as reference. If we have entries without
// updates in the registry, that are older than `started`, we will use `started
// + ttl` to decide if an entry will be removed. This way old entries are not
// removed immediately on startup if the Beat is down for a longer period of
// time.
func gcStore(log *logp.Logger, started time.Time, store *store) {
	log.Debugf("Start store cleanup")
	defer log.Debugf("Done store cleanup")

	states := store.ephemeralStore
	states.mu.Lock()
	defer states.mu.Unlock()

	keys := gcFind(states.table, started, time.Now())
	if len(keys) == 0 {
		log.Debug("No entries to remove were found")
		return
	}

	if err := gcClean(store, keys); err != nil {
		log.Errorf("Failed to remove all entries from the registry: %+v", err)
	}
}

// gcFind searches the store of resources that can be removed. A set of keys to delete is returned.
func gcFind(table map[string]*resource, started, now time.Time) map[string]struct{} {
	keys := map[string]struct{}{}
	for key, resource := range table {
		clean := checkCleanResource(started, now, resource)
		if !clean {
			// do not clean the resource if it is still live or not serialized to the persistent store yet.
			continue
		}
		keys[key] = struct{}{}
	}

	return keys
}

// gcClean removes key value pairs in the removeSet from the store.
// If deletion in the persistent store fails the entry is kept in memory and
// eventually cleaned up later.
func gcClean(store *store, removeSet map[string]struct{}) error {
	for key := range removeSet {
		if err := store.persistentStore.Remove(key); err != nil {
			return err
		}
		delete(store.ephemeralStore.table, key)
	}
	return nil
}

// checkCleanResource returns true for a key-value pair is assumed to be old,
// if is not in use and there are no more pending updates that still need to be
// written to the persistent store anymore.
func checkCleanResource(started, now time.Time, resource *resource) bool {
	if !resource.Finished() {
		return false
	}

	resource.stateMutex.Lock()
	defer resource.stateMutex.Unlock()

	ttl := resource.internalState.TTL
	reference := resource.internalState.Updated
	if started.After(reference) {
		reference = started
	}

	return reference.Add(ttl).Before(now) && resource.stored
}
