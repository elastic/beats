// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
)

const (
	statePrefix      = "filebeat::aws-cloudwatch::state::"
	inputGroupArn    = "groupArn"
	inputGroupName   = "groupName"
	inputGroupPrefix = "groupPrefix"
)

type StorableState struct {
	LastSyncEpoch int64 `json:"last_sync_epoch" struct:"last_sync_epoch"`
}

type stateHandler struct {
	lock  sync.Mutex
	store *statestore.Store
}

func createStateHandler(store statestore.States) (*stateHandler, error) {
	st, err := store.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("error accessing persistence store: %w", err)
	}

	return &stateHandler{store: st}, nil
}

func (s *stateHandler) GetState(forCfg config) (StorableState, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var ss StorableState

	id, err := generateID(forCfg)
	if err != nil {
		return ss, err
	}

	got, err := s.store.Has(id)
	if err != nil {
		return StorableState{}, err
	}

	if !got {
		// Set to Epoch Zero, which is as if start from beginning
		return StorableState{LastSyncEpoch: 0}, nil
	}

	err = s.store.Get(id, &ss)
	if err != nil {
		return StorableState{}, err
	}

	return ss, nil
}

func (s *stateHandler) StoreState(forCfg config, ss StorableState) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	id, err := generateID(forCfg)
	if err != nil {
		return err
	}

	err = s.store.Set(id, ss)
	if err != nil {
		return err
	}

	return nil
}

func (s *stateHandler) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.store.Close()
}

func generateID(forCfg config) (string, error) {
	// first check group ARN
	if forCfg.LogGroupARN != "" {
		return fmt.Sprintf("%s%s::%s", statePrefix, inputGroupArn, forCfg.LogGroupARN), nil
	}

	// then fallback to log group name
	if forCfg.LogGroupName != "" {
		return fmt.Sprintf("%s%s::%s::%s", statePrefix, inputGroupName, forCfg.LogGroupName, forCfg.RegionName), nil
	}

	// finally fallback to log group prefix
	if forCfg.LogGroupNamePrefix != "" {
		return fmt.Sprintf("%s%s::%s::%s", statePrefix, inputGroupPrefix, forCfg.LogGroupNamePrefix, forCfg.RegionName), nil
	}

	return "", fmt.Errorf("incorrect configurations received, missing log_group_arn, log_group_name and log_group_name_prefix properties")
}
