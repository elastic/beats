// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"fmt"
	"sync"

	"github.com/zyedidia/generic/heap"

	"github.com/elastic/beats/v9/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	statePrefix      = "filebeat::aws-cloudwatch::state::"
	inputGroupArn    = "groupArn"
	inputGroupName   = "groupName"
	inputGroupPrefix = "groupPrefix"
)

type storableState struct {
	LastSyncEpoch int64 `json:"last_sync_epoch" struct:"last_sync_epoch"`
}

type tracker struct {
	timeStamp int64
	count     int
}

// stateHandler wraps state handling.
// It allows to get stored state, track state updates and store the most appropriate state.
type stateHandler struct {
	id    string
	store *statestore.Store
	log   *logp.Logger

	registerReceiver chan tracker
	completeReceiver chan int64
	shutdown         chan struct{}

	lock sync.Mutex
}

func newStateHandler(log *logp.Logger, cfg config, store statestore.States) (*stateHandler, error) {
	id, err := generateID(cfg)
	if err != nil {
		return nil, err
	}
	st, err := store.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("error accessing persistence store: %w", err)
	}

	sh := &stateHandler{
		id:               id,
		store:            st,
		log:              log,
		registerReceiver: make(chan tracker),
		completeReceiver: make(chan int64),
		shutdown:         make(chan struct{}),
	}

	go sh.backgroundRunner()
	return sh, nil
}

// GetState returns the previously stored state if available.
// Returned state corresponds to the id generated based on configurations.
func (s *stateHandler) GetState() (storableState, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var ss storableState
	got, err := s.store.Has(s.id)
	if err != nil {
		return storableState{}, err
	}

	if !got {
		// Set to Epoch Zero, which is as if start from beginning
		return storableState{LastSyncEpoch: 0}, nil
	}

	err = s.store.Get(s.id, &ss)
	if err != nil {
		return storableState{}, err
	}

	return ss, nil
}

// WorkRegister accepts work identified through timestamp and amount of work.
func (s *stateHandler) WorkRegister(timestamp int64, workCount int) {
	s.registerReceiver <- tracker{
		timeStamp: timestamp,
		count:     workCount,
	}
}

// WorkComplete accepts an individual work tracked at the given timestamp.
func (s *stateHandler) WorkComplete(timestamp int64) {
	select {
	case s.completeReceiver <- timestamp:
	case <-s.shutdown: // Make sure to not block during a shutdown
	}
}

// backgroundRunner tracks registered work and completed work.
// It stores the oldest tracked work once all work for corresponding timestamp is complete.
func (s *stateHandler) backgroundRunner() {
	trackingMap := map[int64]*tracker{}
	bHeap := heap.New[*tracker](func(a, b *tracker) bool {
		return a.timeStamp < b.timeStamp
	})

	for {
		select {
		case <-s.shutdown:
			return
		case r := <-s.registerReceiver:
			trackingMap[r.timeStamp] = &r
			bHeap.Push(&r)
		case cmp := <-s.completeReceiver:
			// reduce tracked work
			got := trackingMap[cmp]
			got.count -= 1

			// check if oldest entry completed and select most recent oldest entry to store
			var toStore *tracker
			for {
				minElement, _ := bHeap.Peek()
				if minElement == nil || minElement.count != 0 {
					break
				}

				toStore, _ = bHeap.Pop()
				delete(trackingMap, toStore.timeStamp)
			}

			if toStore == nil {
				continue
			}

			if err := s.storeState(storableState{LastSyncEpoch: toStore.timeStamp}); err != nil {
				s.log.Errorf("error storing state: %v", err)
			}
		}
	}
}

func (s *stateHandler) storeState(ss storableState) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.store.Set(s.id, ss)
	if err != nil {
		return err
	}

	return nil
}

func (s *stateHandler) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.store.Close()
	close(s.shutdown)
}

// generateID is a helper to derive state registry identifier matching provided configurations.
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
