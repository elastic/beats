// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package framework

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
)

// Simulated state loader, close enough to test logic related to loading state from ES
// without actually using ES
type loaderDB struct {
	keysToState map[string]*monitorstate.State
	mtx         sync.Mutex
}

func newLoaderDB() *loaderDB {
	return &loaderDB{
		keysToState: map[string]*monitorstate.State{},
		mtx:         sync.Mutex{},
	}
}

func (ldb *loaderDB) AddState(sf stdfields.StdMonitorFields, state *monitorstate.State) {
	ldb.mtx.Lock()
	defer ldb.mtx.Unlock()

	key := keyFor(sf)
	ldb.keysToState[key] = state
}

func (ldb *loaderDB) GetState(sf stdfields.StdMonitorFields) *monitorstate.State {
	ldb.mtx.Lock()
	defer ldb.mtx.Unlock()

	key := keyFor(sf)
	found := ldb.keysToState[key]
	return found
}

func keyFor(sf stdfields.StdMonitorFields) string {
	rfid := "default"
	if sf.RunFrom != nil {
		rfid = sf.RunFrom.ID
	}
	return fmt.Sprintf("%s-%s", rfid, sf.ID)
}

func (ldb *loaderDB) StateLoader() monitorstate.StateLoader {
	return func(sf stdfields.StdMonitorFields) (*monitorstate.State, error) {
		return ldb.GetState(sf), nil
	}
}
