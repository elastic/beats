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
	// ShouldSkipObject determines if an object should be skipped based on state validation.
	ShouldSkipObject(state state, isStateValid func(*logp.Logger, state) bool) bool

	// GetStateID returns the appropriate state ID for the given state.
	GetStateID(state state) string
}

// newPollingStrategy creates the appropriate polling strategy based on configuration flag.
func newPollingStrategy(lexicographicalOrdering bool, log *logp.Logger) pollingStrategy {
	if lexicographicalOrdering {
		return newLexicographicalPollingStrategy(log)
	}
	return newNormalPollingStrategy(log)
}

// normalPollingStrategy implements the default (non-lexicographical) polling behavior.
// In this mode:
// - All objects are listed from the beginning each poll cycle
// - ShouldSkipObject - skips objects that don't pass the validity filter
// - GetStateID - returns the state ID (etag and last modified time for change detection)
type normalPollingStrategy struct {
	log *logp.Logger
}

func newNormalPollingStrategy(log *logp.Logger) pollingStrategy {
	return normalPollingStrategy{log: log}
}

func (s normalPollingStrategy) ShouldSkipObject(state state, isStateValid func(*logp.Logger, state) bool) bool {
	return !isStateValid(s.log, state)
}

func (normalPollingStrategy) GetStateID(state state) string {
	return state.ID()
}

// lexicographicalPollingStrategy implements the lexicographical ordering behavior.
// In this mode:
// - Listing starts from the oldest known key (StartAfter parameter)
// - ShouldSkipObject - doesn't filter by state validity
// - GetStateID - returns the state ID with a lexicographical suffix for isolation
type lexicographicalPollingStrategy struct {
	log *logp.Logger
}

func newLexicographicalPollingStrategy(log *logp.Logger) pollingStrategy {
	return lexicographicalPollingStrategy{log: log}
}

func (lexicographicalPollingStrategy) ShouldSkipObject(state state, isStateValid func(*logp.Logger, state) bool) bool {
	return false
}

func (lexicographicalPollingStrategy) GetStateID(state state) string {
	return state.IDWithLexicographicalOrdering()
}
