// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"context"

	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	ctesting "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/testing"
)

func TestContextProvider(t *testing.T) {
	// first call will have idx of 0
	starting, err := getHostInfo()
	starting["idx"] = 0
	require.NoError(t, err)

	builder := composable.Providers.GetContextProvider("host")
	provider, err := builder(nil)
	require.NoError(t, err)

	hostProvider := provider.(*contextProvider)
	hostProvider.checkTimeout = 100 * time.Millisecond
	hostProvider.fetcher = returnHostMapping()

	ctx, cancel := context.WithCancel(context.Background())
	comm := ctesting.NewContextComm(ctx)
	err = provider.Run(comm)
	require.NoError(t, err)
	require.Equal(t, starting, comm.Current())

	// wait for it to be called again
	var wg sync.WaitGroup
	wg.Add(1)
	comm.CallOnSet(func() {
		wg.Done()
	})
	wg.Wait()
	comm.CallOnSet(nil)
	cancel()

	// next should have been set idx to 1
	next, err := getHostInfo()
	require.NoError(t, err)
	next["idx"] = 1
	assert.Equal(t, next, comm.Current())
}

func returnHostMapping() infoFetcher {
	i := -1
	return func() (map[string]interface{}, error) {
		host, err := getHostInfo()
		if err != nil {
			return nil, err
		}
		i++
		host["idx"] = i
		return host, nil
	}
}
