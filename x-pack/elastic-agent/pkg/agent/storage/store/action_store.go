// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package store

import (
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// ActionStore receives multiples actions to persist to disk, the implementation of the store only
// take care of action policy change every other action are discarded. The store will only keep the
// last good action on disk, we assume that the action is added to the store after it was ACK with
// Fleet. The store is not threadsafe.
// ATTN!!!: THE actionStore is deprecated, please use and extend the stateStore instead. The actionStore will be eventually removed.
type ActionStore struct {
	log    *logger.Logger
	store  storeLoad
	dirty  bool
	action action
}

// NewActionStore creates a new action store.
func NewActionStore(log *logger.Logger, store storeLoad) (*ActionStore, error) {
	// If the store exists we will read it, if any errors is returned we assume we do not have anything
	// persisted and we return an empty store.
	reader, err := store.Load()
	if err != nil {
		return &ActionStore{log: log, store: store}, nil
	}
	defer reader.Close()

	var action ActionPolicyChangeSerializer

	dec := yaml.NewDecoder(reader)
	err = dec.Decode(&action)
	if err == io.EOF {
		return &ActionStore{
			log:   log,
			store: store,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	apc := fleetapi.ActionPolicyChange(action)

	return &ActionStore{
		log:    log,
		store:  store,
		action: &apc,
	}, nil
}

// Add is only taking care of ActionPolicyChange for now and will only keep the last one it receive,
// any other type of action will be silently ignored.
func (s *ActionStore) Add(a action) {
	switch v := a.(type) {
	case *fleetapi.ActionPolicyChange, *fleetapi.ActionUnenroll:
		// Only persist the action if the action is different.
		if s.action != nil && s.action.ID() == v.ID() {
			return
		}
		s.dirty = true
		s.action = a
	}
}

// Save saves actions to backing store.
func (s *ActionStore) Save() error {
	defer func() { s.dirty = false }()
	if !s.dirty {
		return nil
	}

	var reader io.Reader
	if apc, ok := s.action.(*fleetapi.ActionPolicyChange); ok {
		serialize := ActionPolicyChangeSerializer(*apc)

		r, err := yamlToReader(&serialize)
		if err != nil {
			return err
		}

		reader = r
	} else if aun, ok := s.action.(*fleetapi.ActionUnenroll); ok {
		serialize := actionUnenrollSerializer(*aun)

		r, err := yamlToReader(&serialize)
		if err != nil {
			return err
		}

		reader = r
	}

	if reader == nil {
		return fmt.Errorf("incompatible type, expected ActionPolicyChange and received %T", s.action)
	}

	if err := s.store.Save(reader); err != nil {
		return err
	}
	s.log.Debugf("save on disk action policy change: %+v", s.action)
	return nil
}

// Actions returns a slice of action to execute in order, currently only a action policy change is
// persisted.
func (s *ActionStore) Actions() []action {
	if s.action == nil {
		return []action{}
	}

	return []action{s.action}
}

// ActionPolicyChangeSerializer is a struct that adds a YAML serialization, I don't think serialization
// is a concern of the fleetapi package. I went this route so I don't have to do much refactoring.
//
// There are four ways to achieve the same results:
// 1. We create a second struct that map the existing field.
// 2. We add the serialization in the fleetapi.
// 3. We move the actual action type outside of the actual fleetapi package.
// 4. We have two sets of type.
//
// This could be done in a refactoring.
type ActionPolicyChangeSerializer struct {
	ActionID   string                 `yaml:"action_id"`
	ActionType string                 `yaml:"action_type"`
	Policy     map[string]interface{} `yaml:"policy"`
}

// Add a guards between the serializer structs and the original struct.
var _ ActionPolicyChangeSerializer = ActionPolicyChangeSerializer(fleetapi.ActionPolicyChange{})

// actionUnenrollSerializer is a struct that adds a YAML serialization,
type actionUnenrollSerializer struct {
	ActionID   string `yaml:"action_id"`
	ActionType string `yaml:"action_type"`
	IsDetected bool   `yaml:"is_detected"`
}

// Add a guards between the serializer structs and the original struct.
var _ actionUnenrollSerializer = actionUnenrollSerializer(fleetapi.ActionUnenroll{})
