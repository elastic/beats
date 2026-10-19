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

package dns

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestCacheConfigUnpack(t *testing.T) {
	cfg, err := conf.NewConfigFrom(`
type: A
success_cache:
  enabled: false
  capacity.initial: 10
  capacity.max: 20
  min_ttl: 10m
failure_cache:
  enabled: false
  capacity.initial: 5
  capacity.max: 15
  ttl: 30s
`)
	require.NoError(t, err, "parsing the test config")

	c := defaultConfig()
	require.NoError(t, cfg.Unpack(&c), "unpacking the test config")

	assert.Equal(t, cacheSettings{
		Enabled:         false,
		MinTTL:          10 * time.Minute,
		InitialCapacity: 10,
		MaxCapacity:     20,
	}, c.SuccessCache, "success_cache settings were not applied")
	assert.Equal(t, cacheSettings{
		Enabled:         false,
		MinTTL:          time.Minute,
		TTL:             30 * time.Second,
		InitialCapacity: 5,
		MaxCapacity:     15,
	}, c.FailureCache, "failure_cache settings were not applied")
}

func TestCacheConfigUnpackRejectsInvalidSettings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "success capacity.max below the default capacity.initial",
			yaml:    "type: A\nsuccess_cache.capacity.max: 500",
			wantErr: "success_cache.capacity.max must be >= success_cache.capacity.initial",
		},
		{
			name:    "failure ttl not positive",
			yaml:    "type: A\nfailure_cache.ttl: 0s",
			wantErr: "failure_cache.ttl must be > 0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(tc.yaml)
			require.NoError(t, err, "parsing the test config")

			c := defaultConfig()
			assert.ErrorContains(t, cfg.Unpack(&c), tc.wantErr, "invalid cache settings must be rejected at unpack")
		})
	}
}
