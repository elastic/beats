// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// stateStore is a combined agent state storage initially derived from the former actionStore
// and modified to allow persistence of additional agent specific state information.
// The following is the original actionStore implementation description:
// receives multiples actions to persist to disk, the implementation of the store only
// take care of action policy change every other action are discarded. The store will only keep the
// last good action on disk, we assume that the action is added to the store after it was ACK with
// Fleet. The store is not threadsafe.
type stateStore struct {
	log   *logger.Logger
	store storeLoad
	dirty bool
	state stateT
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

	actionStore, err := newActionStore(log, actionDiskStore)
	if err != nil {
		log.Errorf("failed to create action store %s: %v", actionStorePath, err)
		return err
	}

	// no actions stored nothing to migrate
	if len(actionStore.Actions()) == 0 {
		log.Debugf("no actions stored in the action store %s, nothing to migrate", actionStorePath)
		return nil
	}

	stateStore, err := newStateStore(log, stateDiskStore)
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

func newStateStoreWithMigration(log *logger.Logger, actionStorePath, stateStorePath string) (*stateStore, error) {
	err := migrateStateStore(log, actionStorePath, stateStorePath)
	if err != nil {
		return nil, err
	}

	return newStateStore(log, storage.NewDiskStore(stateStorePath))
}

func newStateStore(log *logger.Logger, store storeLoad) (*stateStore, error) {
	// If the store exists we will read it, if any errors is returned we assume we do not have anything
	// persisted and we return an empty store.
	reader, err := store.Load()
	if err != nil {
		return &stateStore{log: log, store: store}, nil
	}
	defer reader.Close()

	var sr stateSerializer

	dec := yaml.NewDecoder(reader)
	err = dec.Decode(&sr)
	if err == io.EOF {
		return &stateStore{
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

	return &stateStore{
		log:   log,
		store: store,
		state: state,
	}, nil
}

// Add is only taking care of ActionPolicyChange for now and will only keep the last one it receive,
// any other type of action will be silently ignored.
func (s *stateStore) Add(a action) {
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
func (s *stateStore) SetAckToken(ackToken string) {
	if s.state.ackToken == ackToken {
		return
	}
	s.dirty = true
	s.state.ackToken = ackToken
}

func (s *stateStore) Save() error {
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
			serialize.Action = &actionSerializer{apc.ActionID, apc.ActionType, nil, &aun.IsDetected}
		} else {
			return fmt.Errorf("incompatible type, expected ActionPolicyChange and received %T", s.state.action)
		}
	}

	reader, err := yamlToReader(&serialize)
	if err != nil {
		return err
	}

	if err := s.store.Save(reader); err != nil {
		return err
	}
	s.log.Debugf("save state on disk : %+v", s.state)
	return nil
}

// Actions returns a slice of action to execute in order, currently only a action policy change is
// persisted.
func (s *stateStore) Actions() []action {
	if s.state.action == nil {
		return []action{}
	}

	return []action{s.state.action}
}

// AckToken return the agent state persisted ack_token
func (s *stateStore) AckToken() string {
	return s.state.ackToken
}

// actionStoreAcker wraps an existing acker and will send any acked event to the action store,
// its up to the action store to decide if we need to persist the event for future replay or just
// discard the event.
type stateStoreActionAcker struct {
	acker fleetAcker
	store *stateStore
}

func (a *stateStoreActionAcker) Ack(ctx context.Context, action fleetapi.Action) error {
	if err := a.acker.Ack(ctx, action); err != nil {
		return err
	}
	a.store.Add(action)
	return a.store.Save()
}

func (a *stateStoreActionAcker) Commit(ctx context.Context) error {
	return a.acker.Commit(ctx)
}

func newStateStoreActionAcker(acker fleetAcker, store *stateStore) *stateStoreActionAcker {
	return &stateStoreActionAcker{acker: acker, store: store}
}

func replayActions(
	log *logger.Logger,
	dispatcher dispatcher,
	acker fleetAcker,
	actions ...action,
) error {
	log.Info("restoring current policy from disk")

	if err := dispatcher.Dispatch(acker, actions...); err != nil {
		return err
	}

	return nil
}
