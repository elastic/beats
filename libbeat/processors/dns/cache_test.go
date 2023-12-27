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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

type stubResolver struct{}

func (r *stubResolver) Lookup(ip string, _ queryType) (*result, error) {
	switch ip {
	case gatewayIP:
		return &result{Data: []string{gatewayName}, TTL: gatewayTTL}, nil
	case gatewayIP + "1":
		return nil, io.ErrUnexpectedEOF
	case gatewayIP + "2":
		return &result{Data: []string{gatewayName}, TTL: 0}, nil
	}
	return nil, &dnsError{"fake lookup returned NXDOMAIN"}
}

func TestCache(t *testing.T) {
	c, err := newLookupCache(
		monitoring.NewRegistry(),
		defaultConfig().cacheConfig,
		&stubResolver{})
	if err != nil {
		t.Fatal(err)
	}

	// Initial success query.
	r, err := c.Lookup(gatewayIP, typePTR)
	if assert.NoError(t, err) {
		assert.EqualValues(t, []string{gatewayName}, r.Data)
		assert.EqualValues(t, gatewayTTL, r.TTL)
		assert.EqualValues(t, 0, c.stats.Hit.Get())
		assert.EqualValues(t, 1, c.stats.Miss.Get())
	}

	// Cached success query.
	r, err = c.Lookup(gatewayIP, typePTR)
	if assert.NoError(t, err) {
		assert.EqualValues(t, []string{gatewayName}, r.Data)
		// TTL counts down while in cache.
		assert.InDelta(t, gatewayTTL, r.TTL, 1)
		assert.EqualValues(t, 1, c.stats.Hit.Get())
		assert.EqualValues(t, 1, c.stats.Miss.Get())
	}

	// Initial failure query (like a dns error response code).
	r, err = c.Lookup(gatewayIP+"0", typePTR)
	if assert.Error(t, err) {
		assert.Nil(t, r)
		assert.EqualValues(t, 1, c.stats.Hit.Get())
		assert.EqualValues(t, 2, c.stats.Miss.Get())
	}

	// Cached failure query.
	r, err = c.Lookup(gatewayIP+"0", typePTR)
	if assert.Error(t, err) {
		assert.Nil(t, r)
		assert.EqualValues(t, 2, c.stats.Hit.Get())
		assert.EqualValues(t, 2, c.stats.Miss.Get())
	}

	// Initial network failure (like I/O timeout).
	r, err = c.Lookup(gatewayIP+"1", typePTR)
	if assert.Error(t, err) {
		assert.Nil(t, r)
		assert.EqualValues(t, 2, c.stats.Hit.Get())
		assert.EqualValues(t, 3, c.stats.Miss.Get())
	}

	// Check for a cache hit for the network failure.
	r, err = c.Lookup(gatewayIP+"1", typePTR)
	if assert.Error(t, err) {
		assert.Nil(t, r)
		assert.EqualValues(t, 3, c.stats.Hit.Get())
		assert.EqualValues(t, 3, c.stats.Miss.Get()) // Cache miss.
	}

	minTTL := defaultConfig().cacheConfig.SuccessCache.MinTTL
	// Initial success returned TTL=0 with MinTTL.
	r, err = c.Lookup(gatewayIP+"2", typePTR)
	if assert.NoError(t, err) {
		assert.EqualValues(t, []string{gatewayName}, r.Data)

		assert.EqualValues(t, minTTL/time.Second, r.TTL)
		assert.EqualValues(t, 3, c.stats.Hit.Get())
		assert.EqualValues(t, 4, c.stats.Miss.Get())

		expectedExpire := time.Now().Add(minTTL).Unix()
		gotExpire := c.success.data[gatewayIP+"2"].expires.Unix()
		assert.InDelta(t, expectedExpire, gotExpire, 1)
	}

	// Cached success from a previous TTL=0 response.
	r, err = c.Lookup(gatewayIP+"2", typePTR)
	if assert.NoError(t, err) {
		assert.EqualValues(t, []string{gatewayName}, r.Data)
		// TTL counts down while in cache.
		assert.InDelta(t, minTTL/time.Second, r.TTL, 1)
		assert.EqualValues(t, 4, c.stats.Hit.Get())
		assert.EqualValues(t, 4, c.stats.Miss.Get())
	}
}
