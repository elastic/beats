// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

// stateRegistryV2 wraps the existing stateRegistry interface for use by
// inputV2. It provides a unified creation path: capacity=0 gives the normal
// (unbounded) registry, capacity>0 gives lexicographical ordering with that
// capacity. Both read existing persisted state in either format on load.
type stateRegistryV2 struct {
	stateRegistry
}

// stateRegistryV2Config holds the parameters for creating a V2 state registry.
type stateRegistryV2Config struct {
	Log       *logp.Logger
	Store     statestore.States
	KeyPrefix string
	// Capacity controls behaviour:
	//   0  = normal (unbounded) registry, no tail tracking
	//   >0 = lexicographical registry with capacity-limited state and tail tracking
	Capacity int
}

// newStateRegistryV2 creates a state registry for the V2 input. It delegates
// to the existing newStateRegistry which loads persisted state (both normal
// and lexicographical formats) and handles cleanup/persistence.
func newStateRegistryV2(cfg stateRegistryV2Config) (*stateRegistryV2, error) {
	lexicographical := cfg.Capacity > 0
	reg, err := newStateRegistry(cfg.Log, cfg.Store, cfg.KeyPrefix, lexicographical, cfg.Capacity)
	if err != nil {
		return nil, err
	}
	return &stateRegistryV2{stateRegistry: reg}, nil
}

// MarkProcessed records that an object has been fully processed (all events
// ACKed). This is the V2 entry point that combines setting Stored=true and
// calling AddState.
func (r *stateRegistryV2) MarkProcessed(bucket, key, etag string, obj s3EventV2) error {
	st := newState(bucket, key, etag, obj.S3.Object.LastModified)
	st.Stored = true
	return r.AddState(st)
}

// MarkFailed records a permanent processing failure for an object. The object
// will not be retried on subsequent polls.
func (r *stateRegistryV2) MarkFailed(bucket, key, etag string, obj s3EventV2) error {
	st := newState(bucket, key, etag, obj.S3.Object.LastModified)
	st.Failed = true
	return r.AddState(st)
}
