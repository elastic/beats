// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package secretstores

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func newTestReader(t *testing.T, fetch func(context.Context, string) (map[string]any, error)) (*Reader[string], *time.Time, *[]time.Duration) {
	t.Helper()
	r := NewReader[string](logptest.NewTestingLogger(t, ""))
	r.Name = "test"
	r.Address = "https://user:pw@store:1"
	r.Noun = "secret"
	r.Describe = func(k string) string { return k }
	r.Fetch = fetch
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var sleeps []time.Duration
	r.Now = func() time.Time { return now }
	r.Sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	return r, &now, &sleeps
}

func TestReaderCachesReadsAndFailures(t *testing.T) {
	reads := map[string]int{}
	denied := errors.New("test: denied")
	r, now, _ := newTestReader(t, func(_ context.Context, k string) (map[string]any, error) {
		reads[k]++
		if k == "denied" {
			return nil, denied
		}
		return map[string]any{"k": k}, nil
	})
	ctx := context.Background()

	for range 2 {
		data, err := r.Read(ctx, "ok")
		require.NoError(t, err, "read must work")
		assert.Equal(t, map[string]any{"k": "ok"}, data, "unexpected data")
		_, err = r.Read(ctx, "denied")
		assert.ErrorIs(t, err, denied, "the failure must be returned")
	}
	assert.Equal(t, 1, reads["ok"], "a read must be cached")
	assert.Equal(t, 1, reads["denied"], "a failure must be cached for its key")

	*now = now.Add(DefaultFailureTTL + time.Second)
	_, err := r.Read(ctx, "denied")
	assert.ErrorIs(t, err, denied, "the failure must be returned")
	assert.Equal(t, 2, reads["denied"], "a failure must expire after the failure TTL")
	assert.Equal(t, 1, reads["ok"], "a read must outlive the failure TTL")

	*now = now.Add(DefaultCacheTTL)
	_, err = r.Read(ctx, "ok")
	require.NoError(t, err, "read must work")
	assert.Equal(t, 2, reads["ok"], "a read must expire after the cache TTL")
}

func TestReaderConnectionFailure(t *testing.T) {
	var reads int
	unavailable := &ConnectionError{Err: errors.New("test: status 503"), Cause: "status 503", Retry: true}
	r, now, sleeps := newTestReader(t, func(_ context.Context, k string) (map[string]any, error) {
		reads++
		return nil, unavailable
	})
	ctx := context.Background()

	_, err := r.Read(ctx, "a")
	assert.ErrorIs(t, err, unavailable, "the last error must be returned")
	assert.Equal(t, len(DefaultBackoff)+1, reads, "every attempt must reach the store")
	assert.Equal(t, DefaultBackoff, *sleeps, "the backoff must be used between attempts")

	before := reads
	_, err = r.Read(ctx, "b")
	require.Error(t, err, "the connection failure must be returned for every key")
	assert.ErrorContains(t, err, "test: https://user:xxxxx@store:1 is unavailable", "the address must be redacted")
	assert.NotContains(t, err.Error(), "pw", "credentials must not be in errors")
	assert.Equal(t, before, reads, "the store must not be contacted while the failure is cached")

	*now = now.Add(DefaultFailureTTL + time.Second)
	_, _ = r.Read(ctx, "b")
	assert.Greater(t, reads, before, "the store must be contacted after the failure expires")
}
