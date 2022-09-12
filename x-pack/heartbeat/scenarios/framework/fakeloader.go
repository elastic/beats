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
	mtx         *sync.Mutex
}

func newLoaderDB() *loaderDB {
	return &loaderDB{
		keysToState: map[string]*monitorstate.State{},
		mtx:         &sync.Mutex{},
	}
}

func loaderDbKey(sf stdfields.StdMonitorFields) string {
	if sf.RunFrom != nil {
		return fmt.Sprintf("%s-%s", sf.ID, sf.RunFrom.ID)
	}
	return sf.ID
}

func (ldb loaderDB) AddState(sf stdfields.StdMonitorFields, state *monitorstate.State) {
	ldb.mtx.Lock()
	defer ldb.mtx.Unlock()

	ldb.keysToState[loaderDbKey(sf)] = state
}

func (ldb loaderDB) GetState(sf stdfields.StdMonitorFields) *monitorstate.State {
	ldb.mtx.Lock()
	defer ldb.mtx.Unlock()

	found := ldb.keysToState[loaderDbKey(sf)]
	return found
}

func (ldb loaderDB) StateLoader() monitorstate.StateLoader {
	return func(sf stdfields.StdMonitorFields) (*monitorstate.State, error) {
		ldb.mtx.Lock()
		defer ldb.mtx.Unlock()

		found := ldb.keysToState[loaderDbKey(sf)]
		return found, nil
	}
}
