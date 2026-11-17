// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/extension"

	"github.com/elastic/beats/v7/x-pack/otel/extension/beatsauthextension"
	"github.com/elastic/beats/v7/x-pack/otel/extension/elasticsearchauth"
)

func newTestBeatsAuthenticator(
	t *testing.T,
	endpoint, user, password string,
) (component.ID, extension.Extension) {

	t.Helper()

	id := component.MustNewID("beatsauth")
	authenticator, err := beatsauthextension.NewFactory().Create(
		t.Context(),
		extension.Settings{
			ID:                id,
			TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		},
		&beatsauthextension.Config{
			Endpoints: []string{endpoint},
			BeatAuthConfig: map[string]any{
				"auth": map[string]any{
					"username": user,
					"password": password,
				},
			},
		})
	require.NoError(t, err, "Beats authenticator must be created")
	require.NoError(
		t,
		authenticator.Start(t.Context(), componenttest.NewNopHost()),
		"Beats authenticator must start",
	)
	t.Cleanup(func() {
		require.NoError(
			t,
			authenticator.Shutdown(t.Context()),
			"Beats authenticator must shut down",
		)
	})

	return id, authenticator
}

// newTestElasticsearchAuthenticator creates and starts an elasticsearchauth
// extension configured with basic authentication.
func newTestElasticsearchAuthenticator(
	t *testing.T,
	endpoint, user, password string,
) (component.ID, extension.Extension) {
	t.Helper()

	cfg := &elasticsearchauth.Config{
		ClientConfig: confighttp.NewDefaultClientConfig(),
		Endpoints:    []string{endpoint},
		User:         user,
		Password:     configopaque.String(password),
	}
	return startTestElasticsearchAuthenticator(t, cfg)
}

// newTestElasticsearchAPIKeyAuthenticator creates and starts an
// elasticsearchauth extension configured with ApiKey authentication and
// optional extra headers.
func newTestElasticsearchAPIKeyAuthenticator(
	t *testing.T,
	endpoint, apiKey string,
	headers map[string]string,
) (component.ID, extension.Extension) {
	t.Helper()

	clientConfig := confighttp.NewDefaultClientConfig()
	for name, value := range headers {
		clientConfig.Headers.Set(name, configopaque.String(value))
	}
	cfg := &elasticsearchauth.Config{
		ClientConfig: clientConfig,
		Endpoints:    []string{endpoint},
		APIKey:       configopaque.String(apiKey),
	}
	return startTestElasticsearchAuthenticator(t, cfg)
}

func startTestElasticsearchAuthenticator(
	t *testing.T,
	cfg *elasticsearchauth.Config,
) (component.ID, extension.Extension) {
	t.Helper()

	id := component.MustNewID("elasticsearchauth")
	require.NoError(t, cfg.Validate(), "Elasticsearch authenticator configuration must be valid")
	authenticator, err := elasticsearchauth.NewFactory().Create(
		t.Context(),
		extension.Settings{
			ID:                id,
			TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		},
		cfg)
	require.NoError(t, err, "Elasticsearch authenticator must be created")
	require.NoError(
		t,
		authenticator.Start(t.Context(), componenttest.NewNopHost()),
		"Elasticsearch authenticator must start",
	)
	t.Cleanup(func() {
		require.NoError(
			t,
			authenticator.Shutdown(t.Context()),
			"Elasticsearch authenticator must shut down",
		)
	})

	return id, authenticator
}

func newTestStorageConfig(authenticatorID component.ID) *Config {
	return &Config{
		Auth: configauth.Config{AuthenticatorID: authenticatorID},
	}
}

type storageTestHost map[component.ID]component.Component

func (h storageTestHost) GetExtensions() map[component.ID]component.Component {
	return h
}
