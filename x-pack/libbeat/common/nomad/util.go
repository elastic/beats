// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	api "github.com/hashicorp/nomad/api"
)

type ClientConfig struct {
	Address   string
	Region    string
	SecretID  string
	Namespace string
}

// NewClient returns a new Nomad client, using the default configuration or the configuration options
// provided through environment variables.
func NewClient(config ClientConfig) (*Client, error) {
	apiConfig := api.DefaultConfig()
	if config.Address != "" {
		apiConfig.Address = config.Address
	}
	if config.Region != "" {
		apiConfig.Region = config.Region
	}
	if config.SecretID != "" {
		apiConfig.SecretID = config.SecretID
	}
	if config.Namespace != "" {
		apiConfig.Namespace = config.Namespace
	}
	return api.NewClient(apiConfig)
}

// StringToPtr returns the pointer to a string
func StringToPtr(str string) *string {
	return &str
}

// BoolToPtr returns the pointer to a boolean
func BoolToPtr(b bool) *bool {
	return &b
}
