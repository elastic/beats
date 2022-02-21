// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type dispatcher interface {
	Dispatch(acker FleetAcker, actions ...action) error
}

type store interface {
	Save(io.Reader) error
}

// FleetAcker is an acker of actions to fleet.
type FleetAcker interface {
	Ack(ctx context.Context, action fleetapi.Action) error
	Commit(ctx context.Context) error
}

type storeLoad interface {
	store
	Load() (io.ReadCloser, error)
}

type action = fleetapi.Action

// StateStore is a combined agent state storage initially derived from the former actionStore
// and modified to allow persistence of additional agent specific state information.
// The following is the original actionStore implementation description:
// receives multiples actions to persist to disk, the implementation of the store only
// take care of action policy change every other action are discarded. The store will only keep the
// last good action on disk, we assume that the action is added to the store after it was ACK with
// Fleet. The store is not threadsafe.
type StateStore struct {
	log   *logger.Logger
	store storeLoad
	dirty bool
	state stateT

	mx sync.RWMutex
}

type stateT struct {
	action   action
	ackToken string
}

// Combined yml serializer for the ActionPolicyChange and ActionUnenroll
type actionSerializer struct {
	ID         string                 `yaml:"action_id"`
	Type       string                 `yaml:"action_type"`
	Policy     map[string]interface{} `yaml:"policy,omitempty"`
	IsDetected *bool                  `yaml:"is_detected,omitempty"`
}

type stateSerializer struct {
	Action   *actionSerializer `yaml:"action,omitempty"`
	AckToken string            `yaml:"ack_token,omitempty"`
}

// NewStateStoreWithMigration creates a new state store and migrates the old one.
func NewStateStoreWithMigration(log *logger.Logger, actionStorePath, stateStorePath string) (*StateStore, error) {
	err := migrateStateStore(log, actionStorePath, stateStorePath)
	if err != nil {
		return nil, err
	}

	return NewStateStore(log, storage.NewDiskStore(stateStorePath))
}

// NewStateStoreActionAcker creates a new state store backed action acker.
func NewStateStoreActionAcker(acker FleetAcker, store *StateStore) *StateStoreActionAcker {
	return &StateStoreActionAcker{acker: acker, store: store}
}

// NewStateStore creates a new state store.
func NewStateStore(log *logger.Logger, store storeLoad) (*StateStore, error) {
	// If the store exists we will read it, if any errors is returned we assume we do not have anything
	// persisted and we return an empty store.
	reader, err := store.Load()
	if err != nil {
		return &StateStore{log: log, store: store}, nil
	}
	defer reader.Close()

	var sr stateSerializer

	dec := yaml.NewDecoder(reader)
	err = dec.Decode(&sr)
	if err == io.EOF {
		return &StateStore{
			log:   log,
			store: store,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	state := stateT{
		ackToken: sr.AckToken,
	}

	if sr.Action != nil {
		if sr.Action.IsDetected != nil {
			state.action = &fleetapi.ActionUnenroll{
				ActionID:   sr.Action.ID,
				ActionType: sr.Action.Type,
				IsDetected: *sr.Action.IsDetected,
			}
		} else {
			state.action = &fleetapi.ActionPolicyChange{
				ActionID:   sr.Action.ID,
				ActionType: sr.Action.Type,
				Policy:     sr.Action.Policy,
			}
		}
	}

	return &StateStore{
		log:   log,
		store: store,
		state: state,
	}, nil
}

func migrateStateStore(log *logger.Logger, actionStorePath, stateStorePath string) (err error) {
	log = log.Named("state_migration")
	actionDiskStore := storage.NewDiskStore(actionStorePath)
	stateDiskStore := storage.NewDiskStore(stateStorePath)

	stateStoreExits, err := stateDiskStore.Exists()
	if err != nil {
		log.With()
		log.Errorf("failed to check if state store %s exists: %v", stateStorePath, err)
		return err
	}

	// do not migrate if the state store already exists
	if stateStoreExits {
		log.Debugf("state store %s already exists", stateStorePath)
		return nil
	}

	actionStoreExits, err := actionDiskStore.Exists()
	if err != nil {
		log.Errorf("failed to check if action store %s exists: %v", actionStorePath, err)
		return err
	}

	// delete the actions store file upon successful migration
	defer func() {
		if err == nil && actionStoreExits {
			err = actionDiskStore.Delete()
			if err != nil {
				log.Errorf("failed to delete action store %s exists: %v", actionStorePath, err)
			}
		}
	}()

	// nothing to migrate if the action store doesn't exists
	if !actionStoreExits {
		log.Debugf("action store %s doesn't exists, nothing to migrate", actionStorePath)
		return nil
	}

	actionStore, err := NewActionStore(log, actionDiskStore)
	if err != nil {
		log.Errorf("failed to create action store %s: %v", actionStorePath, err)
		return err
	}

	// no actions stored nothing to migrate
	if len(actionStore.Actions()) == 0 {
		log.Debugf("no actions stored in the action store %s, nothing to migrate", actionStorePath)
		return nil
	}

	stateStore, err := NewStateStore(log, stateDiskStore)
	if err != nil {
		return err
	}

	// set actions from the action store to the state store
	stateStore.Add(actionStore.Actions()[0])

	err = stateStore.Save()
	if err != nil {
		log.Debugf("failed to save agent state store %s, err: %v", stateStorePath, err)
	}
	return err
}

// Add is only taking care of ActionPolicyChange for now and will only keep the last one it receive,
// any other type of action will be silently ignored.
func (s *StateStore) Add(a action) {
	s.mx.Lock()
	defer s.mx.Unlock()

	switch v := a.(type) {
	case *fleetapi.ActionPolicyChange, *fleetapi.ActionUnenroll:
		// Only persist the action if the action is different.
		if s.state.action != nil && s.state.action.ID() == v.ID() {
			return
		}
		s.dirty = true
		s.state.action = a
	}
}

// SetAckToken set ack token to the agent state
func (s *StateStore) SetAckToken(ackToken string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.state.ackToken == ackToken {
		return
	}
	s.dirty = true
	s.state.ackToken = ackToken
}

// Save saves the actions into a state store.
func (s *StateStore) Save() error {
	s.mx.Lock()
	defer s.mx.Unlock()

	defer func() { s.dirty = false }()
	if !s.dirty {
		return nil
	}

	var reader io.Reader
	serialize := stateSerializer{
		AckToken: s.state.ackToken,
	}

	if s.state.action != nil {
		if apc, ok := s.state.action.(*fleetapi.ActionPolicyChange); ok {
			serialize.Action = &actionSerializer{apc.ActionID, apc.ActionType, apc.Policy, nil}
		} else if aun, ok := s.state.action.(*fleetapi.ActionUnenroll); ok {
			serialize.Action = &actionSerializer{aun.ActionID, aun.ActionType, nil, &aun.IsDetected}
		} else {
			return fmt.Errorf("incompatible type, expected ActionPolicyChange and received %T", s.state.action)
		}
	}

	reader, err := yamlToReader(&serialize)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	r := io.TeeReader(reader, &buf)
	if err := s.store.Save(r); err != nil {
		return err
	}

	bs := buf.Bytes()
	s.log.With("state.yaml", string(bs)).Infof("saved state on disk")

	s.log.Debugf("saved state on disk: %+v", s.state)
	return nil
}

// Actions returns a slice of action to execute in order, currently only a action policy change is
// persisted.
func (s *StateStore) Actions() []action {
	s.mx.RLock()
	defer s.mx.RUnlock()

	if s.state.action == nil {
		return []action{}
	}

	return []action{s.state.action}
}

// AckToken return the agent state persisted ack_token
func (s *StateStore) AckToken() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.state.ackToken
}

// StateStoreActionAcker wraps an existing acker and will send any acked event to the action store,
// its up to the action store to decide if we need to persist the event for future replay or just
// discard the event.
type StateStoreActionAcker struct {
	acker FleetAcker
	store *StateStore
}

// Ack acks action using underlying acker.
// After action is acked it is stored to backing store.
func (a *StateStoreActionAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	if err := a.acker.Ack(ctx, action); err != nil {
		return err
	}
	a.store.Add(action)
	return a.store.Save()
}

// Commit commits acks.
func (a *StateStoreActionAcker) Commit(ctx context.Context) error {
	return a.acker.Commit(ctx)
}

// ReplayActions replays list of actions.
func ReplayActions(
	log *logger.Logger,
	dispatcher dispatcher,
	acker FleetAcker,
	actions ...action,
) error {
	log.Info("restoring current policy from disk")

	if err := dispatcher.Dispatch(acker, actions...); err != nil {
		return err
	}

	return nil
}

func yamlToReader(in interface{}) (io.Reader, error) {
	data, err := yaml.Marshal(in)
	if err != nil {
		return nil, errors.New(err, "could not marshal to YAML")
	}
	return bytes.NewReader(data), nil
}
