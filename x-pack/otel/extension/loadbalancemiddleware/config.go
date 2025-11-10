// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package loadbalancemiddleware

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	Endpoints       []string `mapstructure:"endpoints"`
	Path            string   `mapstructure:"path"`
	Protocol        string   `mapstructure:"protocol"`
	ContinueOnError bool     `mapstructure:"continue_on_error"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}
