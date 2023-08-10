// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package framework

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
)

var id atomic.Int64

// Simulated state loader, close enough to test logic related to loading state from ES
// without actually using ES
type loaderDB struct {
	keysToState map[string]*monitorstate.State
	mtx         *sync.Mutex
	lastTime    time.Time
	id          string
}

func (ldb *loaderDB) String() string {
	return ldb.id

}

func newLoaderDB() *loaderDB {
	fmt.Println("LDB NEW")
	return &loaderDB{
		keysToState: map[string]*monitorstate.State{},
		mtx:         &sync.Mutex{},
		id:          fmt.Sprintf("I%d", id.Add(1)),
	}
}

func (ldb loaderDB) AddState(sf stdfields.StdMonitorFields, state *monitorstate.State) {
	fmt.Println("LDB ADDSTATE", sf.ID, state)
	ldb.mtx.Lock()
	defer ldb.mtx.Unlock()

	ldb.lastTime = time.Now()

	key := keyFor(sf)
	fmt.Println("LDB ADDKEY   ", ldb.id, key, sf.RunFrom, ldb.lastTime)
	ldb.keysToState[key] = state

	fmt.Println("LDB ADDSTATE ADDED", ldb.id, ldb.keysToState, sf.ID, state)
}

func (ldb loaderDB) GetState(sf stdfields.StdMonitorFields) *monitorstate.State {
	ldb.mtx.Lock()
	defer ldb.mtx.Unlock()

	key := keyFor(sf)
	found := ldb.keysToState[key]
	fmt.Println("LDB STATELOAD", ldb.id, key, found, ldb.keysToState, ldb.lastTime)
	return found
}

func keyFor(sf stdfields.StdMonitorFields) string {
	return fmt.Sprintf("%s-%s", sf.RunFrom.ID, sf.ID)
}

func (ldb loaderDB) StateLoader() monitorstate.StateLoader {
	return func(sf stdfields.StdMonitorFields) (*monitorstate.State, error) {
		return ldb.GetState(sf), nil

	}
}
