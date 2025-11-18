// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	BeatAuthConfig  map[string]interface{} `mapstructure:",remain"`
	ContinueOnError bool                   `mapstructure:"continue_on_error"`
}

<<<<<<< HEAD
=======
type BeatsAuthConfig struct {
	Kerberos  *kerberos.Config                 `config:"kerberos"`
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

>>>>>>> d4ec03346 (Add string method to kerberos AuthType (#47687))
func createDefaultConfig() component.Config {
	return &Config{}
}
