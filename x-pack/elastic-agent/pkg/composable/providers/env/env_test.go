// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package env

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	ctesting "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/testing"
)

func TestContextProvider(t *testing.T) {
	builder, _ := composable.Providers.GetContextProvider("env")
	provider, err := builder(nil, nil)
	require.NoError(t, err)

	comm := ctesting.NewContextComm(context.Background())
	err = provider.Run(comm)
	require.NoError(t, err)

	assert.Equal(t, getEnvMapping(), comm.Current())
}
