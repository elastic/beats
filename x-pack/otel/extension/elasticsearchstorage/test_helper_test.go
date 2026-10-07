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
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/extension"

	"github.com/elastic/beats/v7/x-pack/otel/extension/beatsauthextension"
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

func newTestStorageConfig(authenticatorID component.ID) *Config {
	return &Config{
		Auth: configoptional.Some(configauth.Config{AuthenticatorID: authenticatorID}),
	}
}

type storageTestHost map[component.ID]component.Component

func (h storageTestHost) GetExtensions() map[component.ID]component.Component {
	return h
}
