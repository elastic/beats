// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var _ Input = &testInput{}

type testInput struct {
	name   string
	testFn func(testCtx v2.TestContext) error
	runFn  func(runCtx v2.Context, store *Store, client beat.Client) error
}

func (n *testInput) Name() string {
	return n.name
}

func (n *testInput) Test(testCtx v2.TestContext) error {
	if n.testFn != nil {
		return n.testFn(testCtx)
	}

	return nil
}

func (n *testInput) Run(runCtx v2.Context, store *Store, client beat.Client) error {
	if n.runFn != nil {
		return n.runFn(runCtx, store, client)
	}

	return nil
}

var _ beat.Pipeline = &testPipeline{}

type testPipeline struct {
}

func (t testPipeline) ConnectWith(_ beat.ClientConfig) (beat.Client, error) {
	return &testClient{}, nil
}

func (t testPipeline) Connect() (beat.Client, error) {
	return &testClient{}, nil
}

var _ beat.Client = &testClient{}

type testClient struct {
}

func (c *testClient) Publish(_ beat.Event) {

}

func (c *testClient) PublishAll(_ []beat.Event) {

}

func (c *testClient) Close() error {
	return nil
}

func TestInput_Name(t *testing.T) {
	name := "testInput"
	inp := input{
		managedInput: &testInput{
			name: name,
		},
	}

	require.Equal(t, name, inp.Name())
}

func TestInput_Test(t *testing.T) {
	t.Run("test-ok", func(t *testing.T) {
		t.Parallel()

		called := false
		inp := input{
			managedInput: &testInput{
				testFn: func(testCtx v2.TestContext) error {
					called = true
					return nil
				},
			},
		}

		err := inp.Test(v2.TestContext{Logger: logp.L()})
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("test-err", func(t *testing.T) {
		t.Parallel()

		called := false
		inp := input{
			managedInput: &testInput{
				testFn: func(testCtx v2.TestContext) error {
					called = true

					return errors.New("test error")
				},
			},
		}

		err := inp.Test(v2.TestContext{Logger: logp.L()})
		require.ErrorContains(t, err, "test error")
		require.True(t, called)
	})

	t.Run("test-panic", func(t *testing.T) {
		t.Parallel()

		called := false
		inp := input{
			managedInput: &testInput{
				testFn: func(testCtx v2.TestContext) error {
					called = true

					panic("test panic")
				},
			},
		}

		err := inp.Test(v2.TestContext{Logger: logp.L()})
		require.ErrorContains(t, err, "test panic")
		require.True(t, called)
	})
}

func TestInput_Run(t *testing.T) {
	tmpDataDir := t.TempDir()

	paths.Paths = &paths.Path{Data: tmpDataDir}

	t.Run("run-ok", func(t *testing.T) {
		called := false
		inp := input{
			managedInput: &testInput{
				runFn: func(inputCtx v2.Context, store *Store, client beat.Client) error {
					called = true
					return nil
				},
			},
		}

		err := inp.Run(
			v2.Context{
				Logger:      logp.L(),
				ID:          "testInput",
				Cancelation: context.Background(),
			},
			&testPipeline{})

		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("run-err", func(t *testing.T) {
		called := false
		inp := input{
			managedInput: &testInput{
				runFn: func(inputCtx v2.Context, store *Store, client beat.Client) error {
					called = true
					return errors.New("test error")
				},
			},
		}

		err := inp.Run(
			v2.Context{
				Logger:      logp.L(),
				ID:          "testInput",
				Cancelation: context.Background(),
			},
			&testPipeline{})

		require.ErrorContains(t, err, "test error")
		require.True(t, called)
	})

	t.Run("run-panic", func(t *testing.T) {
		called := false
		inp := input{
			managedInput: &testInput{
				runFn: func(inputCtx v2.Context, store *Store, client beat.Client) error {
					called = true
					panic("test panic")
				},
			},
		}

		err := inp.Run(
			v2.Context{
				Logger:      logp.L(),
				ID:          "testInput",
				Cancelation: context.Background(),
			},
			&testPipeline{})

		require.ErrorContains(t, err, "test panic")
		require.True(t, called)
	})
}
