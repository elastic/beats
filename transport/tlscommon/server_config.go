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
	"fmt"

	"github.com/elastic/elastic-agent-libs/config"
)

// ServerConfig defines the user configurable tls options for any TCP based service.
type ServerConfig struct {
	Enabled          *bool               `config:"enabled" yaml:"enabled,omitempty"`
	VerificationMode TLSVerificationMode `config:"verification_mode" yaml:"verification_mode,omitempty"` // one of 'none', 'full', 'strict', 'certificate'
	Versions         []TLSVersion        `config:"supported_protocols" yaml:"supported_protocols,omitempty"`
	CipherSuites     []CipherSuite       `config:"cipher_suites" yaml:"cipher_suites,omitempty"`
	CAs              []string            `config:"certificate_authorities" yaml:"certificate_authorities,omitempty"`
	Certificate      CertificateConfig   `config:",inline" yaml:",inline"`
	CurveTypes       []tlsCurveType      `config:"curve_types" yaml:"curve_types,omitempty"`
	ClientAuth       *TLSClientAuth      `config:"client_authentication" yaml:"client_authentication,omitempty"` //`none`, `optional` or `required`
	CASha256         []string            `config:"ca_sha256" yaml:"ca_sha256,omitempty"`
}

// LoadTLSServerConfig tranforms a ServerConfig into a `tls.Config` to be used directly with golang
// network types.
func LoadTLSServerConfig(config *ServerConfig) (*TLSConfig, error) {
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

	cipherSuites := make([]uint16, len(config.CipherSuites))
	for idx, suite := range config.CipherSuites {
		cipherSuites[idx] = uint16(suite)
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

	clientAuth := TLSClientAuthNone
	if config.ClientAuth != nil {
		clientAuth = *config.ClientAuth
	}

	// return config if no error occurred
	return &TLSConfig{
		Versions:         config.Versions,
		Verification:     config.VerificationMode,
		Certificates:     certs,
		ClientCAs:        cas,
		CipherSuites:     config.CipherSuites,
		CurvePreferences: curves,
		ClientAuth:       tls.ClientAuthType(clientAuth),
		CASha256:         config.CASha256,
	}, nil
}

// Unpack unpacks the TLS Server configuration.
func (c *ServerConfig) Unpack(cfg config.C) error {
	const clientAuthKey = "client_authentication"
	const ca = "certificate_authorities"

	// When we have explicitly defined the `certificate_authorities` in the configuration we default
	// to `required` for the `client_authentication`, when CA is not defined we should set to `none`.
	if cfg.HasField(ca) && !cfg.HasField(clientAuthKey) {
		err := cfg.SetString(clientAuthKey, -1, "required")
		if err != nil {
			return fmt.Errorf("failed to set client_authentication to required: %w", err)
		}
	}
	type serverCfg ServerConfig
	var sCfg serverCfg
	if err := cfg.Unpack(&sCfg); err != nil {
		return err
	}
	*c = ServerConfig(sCfg)
	return nil
}

// Validate values the TLSConfig struct making sure certificate sure we have both a certificate and
// a key.
func (c *ServerConfig) Validate() error {
	if c.IsEnabled() {
		// c.Certificate.Validate() ensures that both a certificate and key
		// are specified, or neither are specified. For server-side TLS we
		// require both to be specified.
		if c.Certificate.Certificate == "" {
			return ErrCertificateUnspecified
		}
	}
	return c.Certificate.Validate()
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *ServerConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
