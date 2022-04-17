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

//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package journald

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestConfigIncludeMatches(t *testing.T) {
	verify := func(t *testing.T, yml string) {
		t.Helper()

		c, err := common.NewConfigWithYAML([]byte(yml), "source")
		require.NoError(t, err)

		conf := defaultConfig()
		require.NoError(t, c.Unpack(&conf))

		assert.EqualValues(t, "_SYSTEMD_UNIT=foo.service", conf.Matches.OR[0].Matches[0].String())
		assert.EqualValues(t, "_SYSTEMD_UNIT=bar.service", conf.Matches.OR[1].Matches[0].String())
	}

	t.Run("normal", func(t *testing.T) {
		const yaml = `
include_matches:
  or:
  - match: _SYSTEMD_UNIT=foo.service
  - match: _SYSTEMD_UNIT=bar.service
`
		verify(t, yaml)
	})

	t.Run("backwards-compatible", func(t *testing.T) {
		const yaml = `
include_matches:
  - _SYSTEMD_UNIT=foo.service
  - _SYSTEMD_UNIT=bar.service
`

		verify(t, yaml)
	})
}
