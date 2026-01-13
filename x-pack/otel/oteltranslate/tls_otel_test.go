// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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

	t.Run("when ssl.enabled = true", func(t *testing.T) {
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
