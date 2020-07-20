// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	api "github.com/hashicorp/nomad/api"
)

// Default Nomad configuration, reads configuration from environment variables.
var defaultConfig = api.DefaultConfig()

// NewClient returns a new Nomad client, using the default configuration or the configuration options
// provided through environment variables.
func NewClient() (*Client, error) {
	return api.NewClient(defaultConfig)
}

// StringToPtr returns the pointer to a string
func StringToPtr(str string) *string {
	return &str
}

// BoolToPtr returns the pointer to a boolean
func BoolToPtr(b bool) *bool {
	return &b
}
