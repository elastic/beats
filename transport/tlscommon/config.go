// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tlscommon

import (
	"crypto/tls"
	"errors"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Config defines the user configurable options in the yaml file.
type Config struct {
	Enabled              *bool                   `config:"enabled" yaml:"enabled,omitempty"`
	VerificationMode     TLSVerificationMode     `config:"verification_mode" yaml:"verification_mode"` // one of 'none', 'full', 'certificate' and 'strict'
	Versions             []TLSVersion            `config:"supported_protocols" yaml:"supported_protocols,omitempty"`
	CipherSuites         []CipherSuite           `config:"cipher_suites" yaml:"cipher_suites,omitempty"`
	CAs                  []string                `config:"certificate_authorities" yaml:"certificate_authorities,omitempty"`
	Certificate          CertificateConfig       `config:",inline" yaml:",inline"`
	CurveTypes           []TLSCurveType          `config:"curve_types" yaml:"curve_types,omitempty"`
	Renegotiation        TLSRenegotiationSupport `config:"renegotiation" yaml:"renegotiation"`
	CASha256             []string                `config:"ca_sha256" yaml:"ca_sha256,omitempty"`
	CATrustedFingerprint string                  `config:"ca_trusted_fingerprint" yaml:"ca_trusted_fingerprint,omitempty"`
	CertificateReload    CertificateReload       `config:"certificate_reload" yaml:"certificate_reload,omitempty"`
}

// LoadTLSConfig will load a certificate from config with all TLS based keys
// defined. If Certificate and CertificateKey are configured, client authentication
// will be configured. If no CAs are configured, the host CA will be used by go
// built-in TLS support.
func LoadTLSConfig(config *Config, logger *logp.Logger) (*TLSConfig, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	var fail []error
	logFail := func(es ...error) {
		for _, e := range es {
			if e != nil {
				fail = append(fail, e)
			}
		}
	}

	curves := make([]tls.CurveID, len(config.CurveTypes))
	for idx, id := range config.CurveTypes {
		curves[idx] = tls.CurveID(id)
	}

	cas, errs := LoadCertificateAuthorities(config.CAs)
	logFail(errs...)

	var certs []tls.Certificate
	var reloader *CertReloader

	// Skip cert reloading when inline PEMs are used; reloading only makes sense with file paths.
	if config.Certificate.Certificate != "" && config.CertificateReload.IsEnabled() &&
		!IsPEMString(config.Certificate.Certificate) && !IsPEMString(config.Certificate.Key) {
		reloadOpts, err := config.Certificate.reloaderOptions()
		logFail(err)
		if config.CertificateReload.ReloadInterval > 0 {
			reloadOpts = append(reloadOpts, WithReloadInterval(config.CertificateReload.ReloadInterval))
		}
		reloader, err = NewCertReloader(config.Certificate.Certificate, config.Certificate.Key, reloadOpts...)
		logFail(err)
	} else {
		cert, err := LoadCertificate(&config.Certificate)
		logFail(err)
		if cert != nil {
			certs = []tls.Certificate{*cert}
		}
	}

	// fail, if any error occurred when loading certificate files
	if len(fail) != 0 {
		return nil, errors.Join(fail...)
	}

	// return config if no error occurred
	return &TLSConfig{
		Versions:             config.Versions,
		Verification:         config.VerificationMode,
		Certificates:         certs,
		RootCAs:              cas,
		CipherSuites:         config.CipherSuites,
		CurvePreferences:     curves,
		Renegotiation:        tls.RenegotiationSupport(config.Renegotiation),
		CASha256:             config.CASha256,
		CATrustedFingerprint: config.CATrustedFingerprint,
		Logger:               logger,
		certReloader:         reloader,
	}, nil
}

// Validate values the TLSConfig struct making sure certificate sure we have both a certificate and
// a key.
func (c *Config) Validate() error {
	for _, v := range c.Versions {
		if err := v.Validate(); err != nil {
			return err
		}

	}
	for _, cs := range c.CipherSuites {
		if err := cs.Validate(); err != nil {
			return err
		}
	}
	for _, ct := range c.CurveTypes {
		if err := ct.Validate(); err != nil {
			return err
		}
	}
	return c.Certificate.Validate()
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *Config) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
