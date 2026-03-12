// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	ElasticsearchConfig map[string]interface{} `mapstructure:",remain"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func (c *Config) Validate() error {
	return nil
}
