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
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/extension"

	"github.com/elastic/beats/v7/x-pack/otel/extension/elasticsearchauth"
)

func newTestElasticsearchAuthenticator(
	t *testing.T,
	endpoint, user, password string,
) (component.ID, extension.Extension) {

	t.Helper()

	id := component.MustNewID("elasticsearchauth")
	authenticator, err := elasticsearchauth.NewFactory().Create(
		t.Context(),
		extension.Settings{ID: id},
		&elasticsearchauth.Config{
			Endpoints: []string{endpoint},
			User:      user,
			Password:  configopaque.String(password),
		})
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
		Auth: configoptional.Some(configauth.Config{AuthenticatorID: authenticatorID}),
	}
}

type storageTestHost map[component.ID]component.Component

func (h storageTestHost) GetExtensions() map[component.ID]component.Component {
	return h
}
