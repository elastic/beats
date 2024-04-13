// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"strings"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/statestore"
)

const (
	awsS3ObjectStatePrefix = "filebeat::aws-s3::state::"
	awsS3WriteCommitPrefix = "filebeat::aws-s3::writeCommit::"
)

type listingInfo struct {
	totObjects int

	mu            sync.Mutex
	storedObjects int
	errorObjects  int
	finalCheck    bool
}

// states handles list of s3 object state. One must use newStates to instantiate a
// file states registry. Using the zero-value is not safe.
type states struct {
	sync.RWMutex

	log *logp.Logger

	// states store, keyed by ID
	states map[string]state

	listingIDs        map[string]struct{}
	listingInfo       *sync.Map
	statesByListingID map[string][]state
}

// newStates generates a new states registry.
func newStates(ctx v2.Context) *states {
	return &states{
		log:               ctx.Logger.Named("states"),
		states:            map[string]state{},
		listingInfo:       new(sync.Map),
		listingIDs:        map[string]struct{}{},
		statesByListingID: map[string][]state{},
	}
}

func (s *states) MustSkip(state state, store *statestore.Store) bool {
	if !s.IsNew(state) {
		s.log.Debugw("not new state in must skip", "state", state)
		return true
	}

	previousState := s.FindPrevious(state)

	// status is forgotten. if there is no previous state and
	// the state.LastModified is before the last cleanStore
	// write commit we can remove
	var commitWriteState commitWriteState
	err := store.Get(awsS3WriteCommitPrefix+state.Bucket+state.ListPrefix, &commitWriteState)
	if err == nil && previousState.IsEmpty() &&
		(state.LastModified.Before(commitWriteState.Time) || state.LastModified.Equal(commitWriteState.Time)) {
		s.log.Debugw("state.LastModified older than writeCommitState in must skip", "state", state, "commitWriteState", commitWriteState)
		return true
	}

	// the previous state is stored or has error: let's skip
	if !previousState.IsEmpty() && previousState.IsProcessed() {
		s.log.Debugw("previous state is stored or has error", "state", state)
		return true
	}

	return false
}

func (s *states) Delete(id string) {
	s.Lock()
	defer s.Unlock()

	delete(s.states, id)
}

// IsListingFullyStored check if listing if fully stored
// After first time the condition is met it will always return false
func (s *states) IsListingFullyStored(listingID string) bool {
	info, ok := s.listingInfo.Load(listingID)
	if !ok {
		return false
	}
	listingInfo, ok := info.(*listingInfo)
	if !ok {
		return false
	}

	listingInfo.mu.Lock()
	defer listingInfo.mu.Unlock()
	if listingInfo.finalCheck {
		return false
	}

	listingInfo.finalCheck = (listingInfo.storedObjects + listingInfo.errorObjects) == listingInfo.totObjects

	if (listingInfo.storedObjects + listingInfo.errorObjects) > listingInfo.totObjects {
		s.log.Warnf("unexepected mixmatch between storedObjects (%d), errorObjects (%d) and totObjects (%d)",
			listingInfo.storedObjects, listingInfo.errorObjects, listingInfo.totObjects)
	}

	return listingInfo.finalCheck
}

// AddListing add listing info
func (s *states) AddListing(listingID string, listingInfo *listingInfo) {
	s.Lock()
	defer s.Unlock()
	s.listingIDs[listingID] = struct{}{}
	s.listingInfo.Store(listingID, listingInfo)
}

// DeleteListing delete listing info
func (s *states) DeleteListing(listingID string) {
	s.Lock()
	defer s.Unlock()
	delete(s.listingIDs, listingID)
	delete(s.statesByListingID, listingID)
	s.listingInfo.Delete(listingID)
}

// Update updates a state. If previous state didn't exist, new one is created
func (s *states) Update(newState state, listingID string) {
	s.Lock()
	defer s.Unlock()

	s.states[newState.ID] = newState

	if listingID == "" || !newState.IsProcessed() {
		return
	}

	// here we increase the number of stored object
	info, ok := s.listingInfo.Load(listingID)
	if !ok {
		return
	}
	listingInfo, ok := info.(*listingInfo)
	if !ok {
		return
	}

	listingInfo.mu.Lock()

	if newState.Stored {
		listingInfo.storedObjects++
	}

	if newState.Error {
		listingInfo.errorObjects++
	}

	listingInfo.mu.Unlock()

	if _, ok := s.statesByListingID[listingID]; !ok {
		s.statesByListingID[listingID] = make([]state, 0)
	}

	s.statesByListingID[listingID] = append(s.statesByListingID[listingID], newState)
}

// FindPrevious lookups a registered state, that matching the new state.
// Returns a zero-state if no match is found.
func (s *states) FindPrevious(newState state) state {
	s.RLock()
	defer s.RUnlock()
	return s.states[newState.ID]
}

// FindPreviousByID lookups a registered state, that matching the id.
// Returns a zero-state if no match is found.
func (s *states) FindPreviousByID(id string) state {
	s.RLock()
	defer s.RUnlock()
	return s.states[id]
}

func (s *states) IsNew(state state) bool {
	s.RLock()
	defer s.RUnlock()
	oldState, exists := s.states[state.ID]
	return !exists || !oldState.IsEqual(&state)
}

// GetStates creates copy of the file states.
func (s *states) GetStates() []state {
	s.RLock()
	defer s.RUnlock()

	newStates := make([]state, 0, len(s.states))
	for _, state := range s.states {
		newStates = append(newStates, state)
	}

	return newStates
}

// GetListingIDs return a of the listing IDs
func (s *states) GetListingIDs() []string {
	s.RLock()
	defer s.RUnlock()
	listingIDs := make([]string, 0, len(s.listingIDs))
	for listingID := range s.listingIDs {
		listingIDs = append(listingIDs, listingID)
	}

	return listingIDs
}

// GetStatesByListingID return a copy of the states by listing ID
func (s *states) GetStatesByListingID(listingID string) []state {
	s.RLock()
	defer s.RUnlock()

	if _, ok := s.statesByListingID[listingID]; !ok {
		return nil
	}

	newStates := make([]state, len(s.statesByListingID[listingID]))
	copy(newStates, s.statesByListingID[listingID])
	return newStates
}

func (s *states) readStatesFrom(store *statestore.Store) error {
	var states []state

	err := store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, awsS3ObjectStatePrefix) {
			return true, nil
		}

		// try to decode. Ignore faulty/incompatible values.
		var st state
		if err := dec.Decode(&st); err != nil {
			// XXX: Do we want to log here? In case we start to store other
			// state types in the registry, then this operation will likely fail
			// quite often, producing some false-positives in the logs...
			return false, err
		}

		st.ID = key[len(awsS3ObjectStatePrefix):]
		states = append(states, st)
		return true, nil
	})
	if err != nil {
		return err
	}

	states = fixStates(states)

	for _, state := range states {
		s.Update(state, "")
	}

	return nil
}

// fixStates cleans up the registry states when updating from an older version
// of filebeat potentially writing invalid entries.
func fixStates(states []state) []state {
	if len(states) == 0 {
		return states
	}

	// we use a map of states here, so to identify and merge duplicate entries.
	idx := map[string]*state{}
	for i := range states {
		state := &states[i]

		old, exists := idx[state.ID]
		if !exists {
			idx[state.ID] = state
		} else {
			mergeStates(old, state) // overwrite the entry in 'old'
		}
	}

	if len(idx) == len(states) {
		return states
	}

	i := 0
	newStates := make([]state, len(idx))
	for _, state := range idx {
		newStates[i] = *state
		i++
	}
	return newStates
}

// mergeStates merges 2 states by trying to determine the 'newer' state.
// The st state is overwritten with the updated fields.
func mergeStates(st, other *state) {
	// update file meta-data. As these are updated concurrently by the
	// inputs, select the newer state based on the update timestamp.
	if st.LastModified.Before(other.LastModified) {
		st.LastModified = other.LastModified
	}
}

func (s *states) writeStates(store *statestore.Store) error {
	for _, state := range s.GetStates() {
		key := awsS3ObjectStatePrefix + state.ID
		if err := store.Set(key, state); err != nil {
			return err
		}
	}
	return nil
}
