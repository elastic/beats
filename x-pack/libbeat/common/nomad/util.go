// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	api "github.com/hashicorp/nomad/api"
)

// Default Nomad configuration, reads configuration from environment variables
var defaultConfig = api.DefaultConfig()

// GetNomadClient returns a new Nomad config, using the default configuration or
// the one passed as a parameter.
func GetNomadClient() (*Client, error) {
	return api.NewClient(defaultConfig)
}
