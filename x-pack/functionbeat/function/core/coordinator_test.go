// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/functionbeat/function/telemetry"
)

var errUnhappy = errors.New("unhappy :(")

type happyRunner struct{}

func (hr *happyRunner) Run(ctx context.Context, _ telemetry.T) error {
	<-ctx.Done()
	return nil
}
func (hr *happyRunner) String() string { return "happyRunner" }

type unhappyRunner struct{}

func (uhr *unhappyRunner) Run(ctx context.Context, _ telemetry.T) error {
	return errUnhappy
}

func (uhr *unhappyRunner) String() string { return "unhappyRunner" }

func TestStart(t *testing.T) {
	t.Run("start the runner", func(t *testing.T) {
		coordinator := NewCoordinator(nil, &happyRunner{}, &happyRunner{})
		ctx, cancel := context.WithCancel(context.Background())
		var err error
		go func() {
			err = coordinator.Run(ctx, telemetry.Ignored())
			assert.NoError(t, err)
		}()
		cancel()
	})

	t.Run("on error shutdown all the runner", func(t *testing.T) {
		coordinator := NewCoordinator(nil, &happyRunner{}, &unhappyRunner{})
		err := coordinator.Run(context.Background(), telemetry.Ignored())
		assert.Error(t, err)
	})

	t.Run("aggregate all errors", func(t *testing.T) {
		coordinator := NewCoordinator(nil, &unhappyRunner{}, &unhappyRunner{})
		err := coordinator.Run(context.Background(), telemetry.Ignored())
		assert.Error(t, err)
	})
}
