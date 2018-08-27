// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "github.com/elastic/beats/libbeat/common"

// Config default configuration for Beatless.
type Config struct {
	Provider *common.ConfigNamespace `config:"provider" validate:"required"`
}

// FunctionConfig minimal configuration from each function.
type FunctionConfig struct {
	Type    string `config:"type"`
	Enabled bool   `config:"enabled"`
}

// DefaultConfig is the default configuration for Beatless.
var DefaultConfig = Config{}
