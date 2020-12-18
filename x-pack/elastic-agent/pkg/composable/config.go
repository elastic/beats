// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable

import "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"

// Config is config for multiple providers.
type Config struct {
	Providers map[string]*config.Config `config:"providers"`
}
