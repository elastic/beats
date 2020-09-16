// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

// Registry serves as a registration point for guesses.
var Registry = Register{
	guesses: make(map[string]tracing.Guesser),
}

// Register stores the registered guesses.
type Register struct {
	guesses map[string]tracing.Guesser
}

// AddGuess registers a new guess.
func (r *Register) AddGuess(guess tracing.Guesser) error {
	if _, found := r.guesses[guess.Name()]; found {
		return fmt.Errorf("guess %s is duplicated", guess.Name())
	}
	r.guesses[guess.Name()] = guess
	return nil
}

// GetList returns a list of registered guesses.
func (r *Register) GetList() (list []tracing.Guesser) {
	for _, guess := range r.guesses {
		list = append(list, guess)
	}
	return list
}
