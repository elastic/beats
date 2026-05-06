// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatsauthextension

import (
	"time"

	"go.opentelemetry.io/collector/component"

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type Config struct {
	BeatAuthConfig  map[string]interface{} `mapstructure:",remain"`
	ContinueOnError bool                   `mapstructure:"continue_on_error"`
}

// CertificateReloadConfig controls periodic re-reading of the TLS client
// certificate and key from disk, allowing certificate rotation without
// restarting the process. It is parsed from the ssl.certificate_reload block;
// when that block is present, Enabled defaults to true.
type CertificateReloadConfig struct {
	// Enabled turns hot reload on or off. A nil value means the user did not
	// set the field; the resolver treats this as true when the
	// certificate_reload block is present, and false otherwise.
	Enabled *bool `config:"enabled"`
	// ReloadInterval is how often the cert/key files are re-read. Defaults
	// to 5s (provided by tlscommon.CertReloader) when zero.
	ReloadInterval time.Duration `config:"reload_interval"`
}

type BeatsAuthConfig struct {
	Kerberos  *kerberos.Config                 `config:"kerberos"`
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

func createDefaultConfig() component.Config {
	return &Config{}
}
