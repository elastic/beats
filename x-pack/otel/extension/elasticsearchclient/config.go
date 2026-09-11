// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchclient

import (
	"go.opentelemetry.io/collector/component"
)

// Config accepts the complete Elasticsearch output configuration. Remaining
// keys are forwarded to eslegclient; this type does not interpret credentials.
type Config struct {
	ElasticsearchConfig map[string]any `mapstructure:",remain"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func (c *Config) Validate() error {
	return nil
}
