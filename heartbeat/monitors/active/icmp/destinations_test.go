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

package icmp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestDestinations(t *testing.T) {
	cfg := conf.MustNewConfigFrom(map[string]any{"hosts": []any{"api.example.com", "192.0.2.1"}})
	got, err := destinations(cfg)
	require.NoError(t, err, "destinations must not fail")
	assert.Equal(t, []string{"api.example.com", "192.0.2.1"}, got, "the hosts must be the destinations")

	_, err = destinations(conf.NewConfig())
	assert.Error(t, err, "a config without hosts must fail")
}
