// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type Config struct {
	BeatAuthConfig  map[string]interface{} `mapstructure:",remain"`
	ContinueOnError bool                   `mapstructure:"continue_on_error"`
}

type esAuthConfig struct {
	Kerberos  *kerberos.Config                 `config:"kerberos"`
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}
