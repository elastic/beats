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

package filestream

import (
<<<<<<< HEAD
=======
	"encoding/json"
	"fmt"
	"strings"
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Run("paths cannot be empty", func(t *testing.T) {
		c := config{Paths: []string{}}
		err := c.Validate()
		require.Error(t, err)
	})

	t.Run("read_until_eof", func(t *testing.T) {
		t.Run("valid config", func(t *testing.T) {
			c, err := conf.NewConfigFrom(`
id: 'some id'
paths: [/foo/bar*]
read_until_eof.enabled: true
read_until_eof.timeout: 1m
close.reader.on_eof: true
close.on_state_change.removed: false
close.on_state_change.renamed: false
`)
			require.NoError(t, err, "could not create config from string")

			got := defaultConfig()
			err = c.Unpack(&got)
			assert.NoError(t, err)
		})

		t.Run("invalid timeout", func(t *testing.T) {
			tcs := map[string]string{
				"is zero":           `0`,
				"smaller than zero": `-1m`,
			}
			for name, cfg := range tcs {
				t.Run(name, func(t *testing.T) {
					c, err := conf.NewConfigFrom(
						fmt.Sprintf(`
id: 'some id'
paths: [/foo/bar*]
read_until_eof.enabled: true
read_until_eof.timeout: %s
`, cfg))
					require.NoError(t, err, "could not create config from string")

					got := defaultConfig()
					err = c.Unpack(&got)
					assert.ErrorContains(t, err, "requires duration >= 1 accessing 'read_until_eof.timeout'")
				})
			}
		})
		t.Run("is compatible with close settings", func(t *testing.T) {
			tcs := map[string]string{
				"close.reader.after_interval":   `close.reader.after_interval: 1m`,
				"close.on_state_change.removed": `close.on_state_change.removed: true`,
				"close.on_state_change.renamed": `close.on_state_change.renamed: true`,
			}
			for name, cfg := range tcs {
				t.Run(name, func(t *testing.T) {
					c, err := conf.NewConfigFrom(
						fmt.Sprintf(`
id: 'some id'
paths: [/foo/bar*]
read_until_eof.enabled: true
%s
`, cfg))
					require.NoError(t, err, "could not create config from string")

					got := defaultConfig()
					err = c.Unpack(&got)
					assert.NoError(t, err,
						"read_until_eof must be compatible with %s", name)
				})
			}
		})
		t.Run("works without close.reader.on_eof", func(t *testing.T) {
			c, err := conf.NewConfigFrom(`
id: 'some id'
paths: [/foo/bar*]
read_until_eof.enabled: true
`)
			require.NoError(t, err, "could not create config from string")

			got := defaultConfig()
			err = c.Unpack(&got)
			assert.NoError(t, err, "read_until_eof should not require close.reader.on_eof")
		})

		t.Run("default is enabled", func(t *testing.T) {
			c, err := conf.NewConfigFrom(`
id: 'some id'
paths: [/foo/bar*]
`)
			require.NoError(t, err, "could not create config from string")

			got := defaultConfig()
			err = c.Unpack(&got)
			require.NoError(t, err)
			assert.True(t, got.ReadUntilEOF.Enabled,
				"read_until_eof.enabled must default to true")
			assert.Equal(t, time.Minute, got.ReadUntilEOF.Timeout,
				"read_until_eof.timeout must default to 1m")
		})

		t.Run("can be disabled", func(t *testing.T) {
			c, err := conf.NewConfigFrom(`
id: 'some id'
paths: [/foo/bar*]
read_until_eof.enabled: false
`)
			require.NoError(t, err, "could not create config from string")

			got := defaultConfig()
			err = c.Unpack(&got)
			require.NoError(t, err)
			assert.False(t, got.ReadUntilEOF.Enabled,
				"read_until_eof.enabled should be false")
		})
	})
}
