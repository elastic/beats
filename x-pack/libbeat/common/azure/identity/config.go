// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identity

// Config holds Azure AD credentials for obtaining OAuth2 tokens.
type Config struct {
	TenantID     string `config:"tenant_id"`
	ClientID     string `config:"client_id"`
	ClientSecret string `config:"client_secret"`

	// Scope is the OAuth2 scope to request, e.g. "api://<app-id>/.default".
	Scope string `config:"scope"`
}

// Enabled returns true when the minimum required fields are set.
func (c *Config) Enabled() bool {
	return c.TenantID != "" && c.ClientID != ""
}
