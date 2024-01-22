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
)

// Config defines the user configurable options in the yaml file.
type Config struct {
	Enabled              *bool                   `config:"enabled" yaml:"enabled,omitempty"`
	VerificationMode     TLSVerificationMode     `config:"verification_mode" yaml:"verification_mode"` // one of 'none', 'full'
	Versions             []TLSVersion            `config:"supported_protocols" yaml:"supported_protocols,omitempty"`
	CipherSuites         []CipherSuite           `config:"cipher_suites" yaml:"cipher_suites,omitempty"`
	CAs                  []string                `config:"certificate_authorities" yaml:"certificate_authorities,omitempty"`
	Certificate          CertificateConfig       `config:",inline" yaml:",inline"`
	CurveTypes           []tlsCurveType          `config:"curve_types" yaml:"curve_types,omitempty"`
	Renegotiation        TLSRenegotiationSupport `config:"renegotiation" yaml:"renegotiation"`
	CASha256             []string                `config:"ca_sha256" yaml:"ca_sha256,omitempty"`
	CATrustedFingerprint string                  `config:"ca_trusted_fingerprint" yaml:"ca_trusted_fingerprint,omitempty"`
}

// LoadTLSConfig will load a certificate from config with all TLS based keys
// defined. If Certificate and CertificateKey are configured, client authentication
// will be configured. If no CAs are configured, the host CA will be used by go
// built-in TLS support.
func LoadTLSConfig(config *Config) (*TLSConfig, error) {
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

	cert, err := LoadCertificate(&config.Certificate)
	logFail(err)

	cas, errs := LoadCertificateAuthorities(config.CAs)
	logFail(errs...)

	// fail, if any error occurred when loading certificate files
	if len(fail) != 0 {
		return nil, errors.Join(fail...)
	}

	certs := make([]tls.Certificate, 0)
	if cert != nil {
		certs = []tls.Certificate{*cert}
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
	}, nil
}

// Validate values the TLSConfig struct making sure certificate sure we have both a certificate and
// a key.
func (c *Config) Validate() error {
	return c.Certificate.Validate()
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *Config) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
