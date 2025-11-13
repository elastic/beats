// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	BeatAuthConfig  map[string]interface{} `mapstructure:",remain"`
	ContinueOnError bool                   `mapstructure:"continue_on_error"`
}

type BeatsAuthConfig struct {
	Transport   httpcommon.HTTPTransportSettings `config:",inline"`
	LoadBalance bool                             `config:"loadbalance"`
	Endpoints   []string                         `config:"hosts"`
	Path        string                           `config:"path"`
	Protocol    string                           `config:"protocol"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}
