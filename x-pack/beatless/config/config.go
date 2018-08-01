// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

// Config default configuration for Beatless.
type Config struct {
}

// DefaultConfig is the default configuration for Beatless.
var DefaultConfig = Config{}
