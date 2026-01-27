// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"github.com/elastic/elastic-agent-libs/logp"
)

// pollingStrategy defines the strategy interface for S3 polling behavior.
// It is added to support normal mode vs lexicographical ordering mode.
type pollingStrategy interface {
	// PrePollSetup is called before each polling run to perform any necessary setup.
	PrePollSetup(log *logp.Logger, registry stateRegistry)

	// GetStartAfterKey returns the S3 key to start listing from (for StartAfter parameter).
	GetStartAfterKey(registry stateRegistry) string

	// ShouldSkipObject determines if an object should be skipped based on state validation.
	ShouldSkipObject(log *logp.Logger, state state, isStateValid func(*logp.Logger, state) bool) bool

	// GetStateID returns the appropriate state ID for the given state.
	GetStateID(state state) string
}

// newPollingStrategy creates the appropriate polling strategy based on configuration flag.
func newPollingStrategy(lexicographicalOrdering bool) pollingStrategy {
	if lexicographicalOrdering {
		return newLexicographicalPollingStrategy()
	}
	return newNormalPollingStrategy()
}

// normalPollingStrategy implements the default (non-lexicographical) polling behavior.
// In this mode:
// - All objects are listed from the beginning each poll cycle
// - PrePollSetup - not required
// - GetStartAfterKey - always returns empty string
// - ShouldSkipObject - skips objects that don't pass the validity filter
// - GetStateID - returns the state ID (etag and last modified time for change detection)
type normalPollingStrategy struct{}

func newNormalPollingStrategy() pollingStrategy {
	return normalPollingStrategy{}
}

func (normalPollingStrategy) PrePollSetup(log *logp.Logger, registry stateRegistry) {
	// No setup needed for normal mode
}

func (normalPollingStrategy) GetStartAfterKey(registry stateRegistry) string {
	// Doesn't use StartAfter - lists from beginning each poll cycle
	return ""
}

func (normalPollingStrategy) ShouldSkipObject(log *logp.Logger, state state, isStateValid func(*logp.Logger, state) bool) bool {
	return !isStateValid(log, state)
}

func (normalPollingStrategy) GetStateID(state state) string {
	return state.ID()
}

// lexicographicalPollingStrategy implements the lexicographical ordering behavior.
// In this mode:
// - Listing starts from the oldest known key (StartAfter parameter)
// - PrePollSetup - no setup needed, heap maintains order automatically
// - GetStartAfterKey - returns the oldest state's key as the starting point for S3 listing
// - ShouldSkipObject - doesn't filter by state validity
// - GetStateID - returns the state ID with a lexicographical suffix for isolation
type lexicographicalPollingStrategy struct{}

func newLexicographicalPollingStrategy() pollingStrategy {
	return lexicographicalPollingStrategy{}
}

func (lexicographicalPollingStrategy) PrePollSetup(log *logp.Logger, registry stateRegistry) {
	// No setup needed - heap maintains order automatically
}

func (lexicographicalPollingStrategy) GetStartAfterKey(registry stateRegistry) string {
	oldestState := registry.GetLeastState()
	if oldestState != nil {
		return oldestState.Key
	}
	return ""
}

func (lexicographicalPollingStrategy) ShouldSkipObject(log *logp.Logger, state state, isStateValid func(*logp.Logger, state) bool) bool {
	return false
}

func (lexicographicalPollingStrategy) GetStateID(state state) string {
	return state.IDWithLexicographicalOrdering()
}
