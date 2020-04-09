// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package log

// Config is a configuration describing log reporter behavior
type Config struct {
	Format Format `config:"format" yaml:"format"`
}

// DefaultLogConfig initiates LogConfig with default values
func DefaultLogConfig() *Config {
	return &Config{
		Format: DefaultFormat,
	}
}
