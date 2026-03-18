// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package akamai

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

type cursorStore struct {
	store *statestore.Store
	key   string
	log   *logp.Logger
}

func newCursorStore(states statestore.States, key string, log *logp.Logger) (*cursorStore, error) {
	store, err := states.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}
	return &cursorStore{store: store, key: key, log: log}, nil
}

func (cs *cursorStore) Load() (cursor, error) {
	var cur cursor
	if err := cs.store.Get(cs.key, &cur); err != nil { //nolint:nilerr // missing key on first run is expected, not a failure
		cs.log.Debugw("no persisted cursor found, starting fresh", "key", cs.key)
		return cursor{}, nil
	}
	cs.log.Infow("loaded persisted cursor",
		"key", cs.key,
		"chain_from", cur.ChainFrom,
		"chain_to", cur.ChainTo,
		"caught_up", cur.CaughtUp,
		"last_offset", cur.LastOffset,
	)
	return cur, nil
}

func (cs *cursorStore) Save(c cursor) error {
	if err := cs.store.Set(cs.key, c); err != nil {
		return fmt.Errorf("failed to persist cursor (key=%s): %w", cs.key, err)
	}
	cs.log.Debugw("cursor persisted",
		"key", cs.key,
		"chain_from", c.ChainFrom,
		"chain_to", c.ChainTo,
		"last_offset", c.LastOffset,
	)
	return nil
}

func (cs *cursorStore) Close() error {
	return cs.store.Close()
}
