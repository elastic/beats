// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process"
)

// ReattachInfo consists of information needed to
// attach to already started process after host restart.
type ReattachInfo struct {
	ExecutionContext ExecutionContext `json:"execution_context"`
	Address          string           `json:"network_address"`
	PID              int              `json:"pid"`
}

// ReattachCollection represents group of processes
// host knows about.
type reattachCollection struct {
	sync.Mutex
	config *Config
}

func newReattachCollection(config *Config) *reattachCollection {
	return &reattachCollection{config: config}
}

func (rc *reattachCollection) addProcess(ctx ExecutionContext, pi *process.Info) error {
	rc.Lock()
	defer rc.Unlock()

	items, err := rc.items()
	if err != nil {
		return err
	}

	// add process into collection
	ri := &ReattachInfo{
		Address:          pi.Address,
		PID:              pi.PID,
		ExecutionContext: ctx,
	}
	items = append(items, ri)

	return rc.save(items)
}

func (rc *reattachCollection) removeProcess(pid int) error {
	rc.Lock()
	defer rc.Unlock()

	items, err := rc.items()
	if err != nil {
		return err
	}

	// remove process from collection
	id := -1
	for i, ri := range items {
		if ri.PID == pid {
			id = i
			break
		}
	}

	if id == -1 {
		return nil
	}

	items = append(items[:id], items[id+1:]...)
	return rc.save(items)
}

func (rc *reattachCollection) items() ([]*ReattachInfo, error) {
	var rr []*ReattachInfo

	f, err := os.Open(rc.config.ReattachCollectionPath)
	if err != nil && !os.IsNotExist(err) {
		// if file is there and we failed to load, error
		return nil, err
	} else if err == nil {
		// if file is there load it
		dec := json.NewDecoder(f)
		if err := dec.Decode(&rr); err != nil {
			return nil, err
		}
	}

	return rr, nil
}

func (rc *reattachCollection) save(items []*ReattachInfo) error {
	f, err := os.OpenFile(rc.config.ReattachCollectionPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	return json.NewEncoder(f).Encode(items)
}
