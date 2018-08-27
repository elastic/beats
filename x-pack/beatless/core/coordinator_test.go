// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errUnhappy = errors.New("unhappy :(")

type happyRunner struct{}

func (hr *happyRunner) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

type unhappyRunner struct{}

func (uhr *unhappyRunner) Run(ctx context.Context) error {
	return errUnhappy
}

func TestStart(t *testing.T) {
	t.Run("start the runner", func(t *testing.T) {
		coordinator := NewCoordinator(nil, []Runner{&happyRunner{}, &happyRunner{}})
		ctx, cancel := context.WithCancel(context.Background())
		var err error
		go func() {
			err = coordinator.Start(ctx)
			assert.NoError(t, err)
		}()
		cancel()
	})

	t.Run("on error shutdown all the runner", func(t *testing.T) {
		coordinator := NewCoordinator(nil, []Runner{&happyRunner{}, &unhappyRunner{}})
		err := coordinator.Start(context.Background())
		assert.Error(t, err)
	})

	t.Run("aggregate all errors", func(t *testing.T) {
		coordinator := NewCoordinator(nil, []Runner{&unhappyRunner{}, &unhappyRunner{}})
		err := coordinator.Start(context.Background())
		assert.Error(t, err)
	})
}
