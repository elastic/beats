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

package oteltranslate

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/config/configtls"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// currently unsupported parameters
// ssl.curve_types
// ssl.ca_sha256
// ssl.ca_trustred_fingerprint

// ssl.supported_protocols -> partially supported
// ssl.restart_on_cert_change.*
// ssl.renegotiation
// ssl.verification_mode: All modes are not distinctly mapped yet
func validateUnsupportedConfig(tlscfg *tlscommon.Config) error {
	if len(tlscfg.CurveTypes) > 0 {
		return errors.New("setting ssl.curve_types is currently not supported")
	}
	if tlscfg.CATrustedFingerprint != "" {
		return errors.New("setting ssl.ca_trusted_fingerprint is currently not supported")
	}
	if len(tlscfg.CASha256) > 0 {
		return errors.New("setting ssl.ca_sha256 is currently not supported")
	}
	return nil
}

// TLSCommonToOTel converts a tlscommon.Config into the OTel configtls.ClientConfig
func TLSCommonToOTel(tlscfg *tlscommon.Config) (map[string]any, error) {
	logger := logp.L().Named("tls-to-otel")
	insecureSkipVerify := false

	if tlscfg == nil {
		return nil, nil
	}

	if !tlscfg.IsEnabled() {
		return map[string]any{
			"insecure": true,
		}, nil
	}

	// throw error if unsupported tls config is passed
	if err := validateUnsupportedConfig(tlscfg); err != nil {
		return nil, err
	}

	// validate the beats config before proceeding
	if err := tlscfg.Validate(); err != nil {
		return nil, err
	}

	// unpacks -> ssl.certificate_authorities
	// The OTel exporter accepts either single CA file or CA string. However,
	// Beats support any combination and number of files and certificates
	// as string, so we read them all and assemble one PEM string with
	// all CA certificates
	var caCerts []string
	for _, ca := range tlscfg.CAs {
		d, err := tlscommon.ReadPEMFile(logger, ca, "")
		if err != nil {
			logger.Errorf("Failed reading CA: %+v", err)
			return nil, err
		}
		caCerts = append(caCerts, string(d))
	}
	// We only include the system certificates if no CA is defined
	includeSystemCACertsPool := len(caCerts) == 0

	var (
		certKeyPem string
		certPem    string
	)

	if tlscfg.Certificate.Key != "" {
		// unpacks ->  ssl.key
		certKeyBytes, err := tlscommon.ReadPEMFile(logger, tlscfg.Certificate.Key, tlscfg.Certificate.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed reading key file: %w", err)
		}
		certKeyPem = string(certKeyBytes)

		// unpacks ->  ssl.certificate
		certBytes, err := tlscommon.ReadPEMFile(logger, tlscfg.Certificate.Certificate, "")
		if err != nil {
			logger.Errorf("Failed reading cert file: %+v", err)
			return nil, fmt.Errorf("failed reading cert file: %w", err)
		}
		certPem = string(certBytes)
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(tlscfg)
	if err != nil {
		return nil, fmt.Errorf("cannot load SSL configuration: %w", err)
	}
	goTLSConfig := tlsConfig.ToConfig()

	// unpacks -> ssl.cipher_suites
	ciphersuites := []string{}
	for _, cs := range goTLSConfig.CipherSuites {
		ciphersuites = append(ciphersuites, tls.CipherSuiteName(cs))
	}

	otelTLSConfig := map[string]any{
		"insecure_skip_verify": insecureSkipVerify, // ssl.verirication_mode,

		// Config
		"include_system_ca_certs_pool": includeSystemCACertsPool,
		"ca_pem":                       strings.Join(caCerts, ""), // ssl.certificate_authorities
		"cert_pem":                     certPem,                   // ssl.certificate
		"key_pem":                      certKeyPem,                // ssl.key
		"cipher_suites":                ciphersuites,              // ssl.cipher_suites
	}

	// For type safety check only
	// the returned valued should match `clienttls.Config` type.
	// it throws an error if non existing key name is used in the returned map structure
	var result configtls.ClientConfig
	d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Squash:      true,
		Result:      &result,
		ErrorUnused: true,
	})

	if err := d.Decode(otelTLSConfig); err != nil {
		return nil, err
	}

	return otelTLSConfig, nil
}
