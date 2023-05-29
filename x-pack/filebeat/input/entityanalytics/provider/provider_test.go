// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRegistry(t *testing.T) {
	err := Register("test", func(logger *logp.Logger) (Provider, error) {
		return nil, errors.New("test error")
	})
	require.NoError(t, err)
	err = Register("test", func(logger *logp.Logger) (Provider, error) {
		return nil, errors.New("test error")
	})
	require.ErrorIs(t, err, ErrExists)

	exists := Has("test")
	require.True(t, exists)
	exists = Has("foobar")
	require.False(t, exists)

	_, err = Get("foobar")
	require.ErrorIs(t, err, ErrNotFound)
	factoryFn, err := Get("test")
	require.NoError(t, err)

	_, err = factoryFn(logp.L())
	require.ErrorContains(t, err, "test error")
}
