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

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
)

// ServerConfig defines the user configurable tls options for any TCP based service.
type ServerConfig struct {
	Enabled          *bool               `config:"enabled"`
	VerificationMode TLSVerificationMode `config:"verification_mode"` // one of 'none', 'full'
	Versions         []TLSVersion        `config:"supported_protocols"`
	CipherSuites     []tlsCipherSuite    `config:"cipher_suites"`
	CAs              []string            `config:"certificate_authorities"`
	Certificate      CertificateConfig   `config:",inline"`
	CurveTypes       []tlsCurveType      `config:"curve_types"`
	ClientAuth       tlsClientAuth       `config:"client_authentication"` //`none`, `optional` or `required`
}

// LoadTLSServerConfig tranforms a ServerConfig into a `tls.Config` to be used directly with golang
// network types.
func LoadTLSServerConfig(config *ServerConfig) (*TLSConfig, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	fail := multierror.Errors{}
	logFail := func(es ...error) {
		for _, e := range es {
			if e != nil {
				fail = append(fail, e)
			}
		}
	}

	var cipherSuites []uint16
	for _, suite := range config.CipherSuites {
		cipherSuites = append(cipherSuites, uint16(suite))
	}

	var curves []tls.CurveID
	for _, id := range config.CurveTypes {
		curves = append(curves, tls.CurveID(id))
	}

	cert, err := LoadCertificate(&config.Certificate)
	logFail(err)

	cas, errs := LoadCertificateAuthorities(config.CAs)
	logFail(errs...)

	// fail, if any error occurred when loading certificate files
	if err = fail.Err(); err != nil {
		return nil, err
	}

	var certs []tls.Certificate
	if cert != nil {
		certs = []tls.Certificate{*cert}
	}

	// return config if no error occurred
	return &TLSConfig{
		Versions:         config.Versions,
		Verification:     config.VerificationMode,
		Certificates:     certs,
		ClientCAs:        cas,
		CipherSuites:     cipherSuites,
		CurvePreferences: curves,
		ClientAuth:       tls.ClientAuthType(config.ClientAuth),
	}, nil
}

// Unpack unpacks the TLS Server configuration.
func (c *ServerConfig) Unpack(cfg common.Config) error {
	const clientAuthKey = "client_authentication"
	const ca = "certificate_authorities"

	// When we have explicitely defined the `certificate_authorities` in the configuration we default
	// to `required` for the `client_authentication`, when CA is not defined we should set to `none`.
	if cfg.HasField(ca) && !cfg.HasField(clientAuthKey) {
		cfg.SetString(clientAuthKey, -1, "required")
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
	return c.Certificate.Validate()
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *ServerConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
