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

package oteltranslate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestTLSCommonToOTel(t *testing.T) {

	logger := logptest.NewTestingLogger(t, "")
	t.Run("when ssl.enabled = false", func(t *testing.T) {
		input := `
ssl:
  enabled: false
`
		cfg := config.MustNewConfigFrom(input)
		got, err := TLSCommonToOTel(cfg, logger)
		require.NoError(t, err)
		want := map[string]any{
			"insecure": true,
		}

		assert.Equal(t, want, got)
	})

	t.Run("when ssl.enabled = false", func(t *testing.T) {
		input := `
ssl:
  enabled: true
`
		cfg := config.MustNewConfigFrom(input)
		got, err := TLSCommonToOTel(cfg, logger)
		require.NoError(t, err)
		want := map[string]any{
			"min_version": "1.2",
			"max_version": "1.3",
		}

		assert.Equal(t, want, got)
	})

	t.Run("when ssl.verification_mode:none", func(t *testing.T) {
		input := `
ssl:
  verification_mode: none
`
		cfg := config.MustNewConfigFrom(input)
		got, err := TLSCommonToOTel(cfg, logger)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"insecure_skip_verify": true,
			"min_version":          "1.2",
			"max_version":          "1.3",
		}, got, "beats to otel ssl mapping")

	})

	t.Run("when unsupported configuration  renegotiation is used", func(t *testing.T) {
		input := `
ssl:
  verification_mode: none
  renegotiation: never
`
		cfg := config.MustNewConfigFrom(input)
		_, err := TLSCommonToOTel(cfg, logger)
		require.Error(t, err)
		require.ErrorIs(t, err, errors.ErrUnsupported)

	})

	t.Run("when unsupported configuration restart_on_cert_change.enabled is used", func(t *testing.T) {
		input := `
ssl:
  verification_mode: none
  restart_on_cert_change.enabled: true
`
		cfg := config.MustNewConfigFrom(input)
		_, err := TLSCommonToOTel(cfg, logger)
		require.Error(t, err)
		require.ErrorIs(t, err, errors.ErrUnsupported)

	})

	t.Run("when unsupported tls version is passed", func(t *testing.T) {
		input := `
ssl:
  verification_mode: none
  supported_protocols: 
   - TLSv1.4
`
		cfg := config.MustNewConfigFrom(input)
		_, err := TLSCommonToOTel(cfg, logger)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid tls version")

	})
}
