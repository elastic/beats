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

package otelcommon

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// TLSCommonToOTel converts a tlscommon.Config into the OTel configtls.ClientConfig
func TLSCommonToOTel(tlscfg *tlscommon.Config) (map[string]any, error) {
	logger := logp.L().Named("tls-to-otel")
	insecureSkipVerify := false

	if tlscfg == nil {
		return nil, nil
	}

	if tlscfg.VerificationMode == tlscommon.VerifyNone {
		insecureSkipVerify = true
	}

	// The OTel exporter accepts either single CA file or CA string. However,
	// Beats support any combination and number of files and certificates
	// as string, so we read them all and assemble one PEM string with
	// all CA certificates
	var caCerts []string
	for _, ca := range tlscfg.CAs {
		d, err := tlscommon.ReadPEMFile(logger, ca, "")
		if err != nil {
			return nil, err
		}
		caCerts = append(caCerts, string(d))
	}

	certKeyBytes, err := tlscommon.ReadPEMFile(logger, tlscfg.Certificate.Key, tlscfg.Certificate.Passphrase)
	certKeyPem := string(certKeyBytes)

	certBytes, err := tlscommon.ReadPEMFile(logger, tlscfg.Certificate.Certificate, "")
	certPem := string(certBytes)

	// We only include the system certificates if no CA is defined
	includeSystemCACertsPool := len(caCerts) == 0

	tlsConfig, err := tlscommon.LoadTLSConfig(tlscfg)
	if err != nil {
		return nil, fmt.Errorf("cannot load SSL configuration: %w", err)
	}
	goTLSConfig := tlsConfig.ToConfig()
	ciphersuites := []string{}
	for _, cs := range goTLSConfig.CipherSuites {
		ciphersuites = append(ciphersuites, tls.CipherSuiteName(cs))
	}

	otelTLSConfig := map[string]any{
		"insecure":             insecureSkipVerify, // ssl.verirication_mode, used for gRPC
		"insecure_skip_verify": insecureSkipVerify, // ssl.verirication_mode, used for HTTPS

		// Config
		"include_system_ca_certs_pool": includeSystemCACertsPool,
		"ca_pem":                       strings.Join(caCerts, ""), // ssl.certificate_authorities
		"cert_pem":                     certPem,                   // ssl.certificate
		"key_pem":                      certKeyPem,                // ssl.key
		"cipher_suites":                ciphersuites,              // ssl.cipher_suites
	}

	return otelTLSConfig, nil
}
