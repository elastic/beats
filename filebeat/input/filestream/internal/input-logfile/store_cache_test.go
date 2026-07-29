// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with this work for
// additional information regarding copyright ownership. Elasticsearch B.V.
// licenses this file to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package input_logfile

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestStoreCache_AcquireHit(t *testing.T) {
	resetStoreCacheForTest()
	t.Cleanup(resetStoreCacheForTest)

	states := createSampleStore(t, nil).WithGCPeriod(time.Minute)
	first, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	second, err := acquireStore(logptest.NewTestingLogger(t, ""), states, "filestream")
	require.NoError(t, err)
	require.Same(t, first, second)

	globalStoreCache.mu.Lock()
	entry := globalStoreCache.entries[states.StoreKey()]
	require.NotNil(t, entry)
	require.Equal(t, storeActive, entry.state)
	require.Equal(t, 2, entry.users)
	require.Same(t, first, entry.store)
	require.Len(t, globalStoreCache.entries, 1)
	globalStoreCache.mu.Unlock()

	releaseAcquiredStore(first)
	releaseAcquiredStore(second)
}

func resetStoreCacheForTest() {
	globalStoreCache.mu.Lock()
	entries := globalStoreCache.entries
	globalStoreCache.entries = make(map[string]*storeCacheEntry)
	globalStoreCache.mu.Unlock()
	for _, entry := range entries {
		if entry.state == storeInitializing {
			entry.initErr = errors.New("store cache reset")
			entry.closeReady()
			entry.closeClosed()
			continue
		}
		if entry.state == storeActive {
			entry.cancel()
			entry.cleanerWg.Wait()
			entry.store.Release()
		}
	}
}
