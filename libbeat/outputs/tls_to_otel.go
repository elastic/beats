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

package outputs

import (
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtls"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// TLSCommonToOtel converts a tlscommon.Config into the OTel configtls.ClientConfig
func TLSCommonToOtel(tlscfg *tlscommon.Config) (configtls.ClientConfig, error) {
	insecureSkipVerify := false
	if tlscfg.VerificationMode == tlscommon.VerifyNone {
		insecureSkipVerify = true
	}

	// The OTel exporter accepts either single CA file or CA string. However
	// Beats support any combination and number of files and certificates
	// as string.
	// TODO (Tiago): Merge all certificates into a single one.
	caFiles := []string{}
	caCerts := []string{}
	for _, ca := range tlscfg.CAs {
		if tlscommon.IsPEMString(ca) {
			caCerts = append(caCerts, ca)
			continue
		}
		caFiles = append(caFiles, ca)
	}

	if len(caFiles) > 1 {
		return configtls.ClientConfig{}, fmt.Errorf("currently a single CA file is supported, got %d", len(caFiles))
	}

	if len(caCerts) > 1 {
		return configtls.ClientConfig{}, fmt.Errorf("currently a single CA certificate is supported, got %d", len(caCerts))
	}

	caFile := ""
	caPem := ""
	if len(caFiles) == 1 {
		caFile = caFiles[0]
	}
	if len(caCerts) == 1 {
		caPem = caCerts[0]
	}

	certFile := tlscfg.Certificate.Certificate
	certPem := ""
	if tlscommon.IsPEMString(tlscfg.Certificate.Certificate) {
		certPem = tlscfg.Certificate.Certificate
		certFile = ""
	}

	certKeyFile := tlscfg.Certificate.Key
	certKeyPem := ""
	if tlscommon.IsPEMString(tlscfg.Certificate.Key) {
		certKeyPem = tlscfg.Certificate.Key
		certKeyFile = ""
	}

	// If custom certificates are set we do not include the system certificates
	includeSystemCACertsPool := (caFile == "") && (caPem == "")

	tlsConfig, err := tlscommon.LoadTLSConfig(tlscfg)
	if err != nil {
		return configtls.ClientConfig{}, fmt.Errorf("cannot load SSL configuration: %w", err)
	}
	goTLSConfig := tlsConfig.ToConfig()
	ciphersuites := []string{}
	for _, cs := range goTLSConfig.CipherSuites {
		ciphersuites = append(ciphersuites, tls.CipherSuiteName(cs))
	}

	otelTLSConfig := configtls.ClientConfig{
		Insecure:           insecureSkipVerify, // ssl.verirication_mode, used for gRPC
		InsecureSkipVerify: insecureSkipVerify, // ssl.verirication_mode, used for HTTPS
		Config: configtls.Config{
			IncludeSystemCACertsPool: includeSystemCACertsPool,
			CAFile:                   caFile,                          // ssl.certificate_authorities
			CAPem:                    configopaque.String(caPem),      // ssl.certificate_authorities
			CertFile:                 certFile,                        // ssl.certificate
			CertPem:                  configopaque.String(certPem),    // ssl.certificate
			KeyFile:                  certKeyFile,                     // ssl.key
			KeyPem:                   configopaque.String(certKeyPem), // ssl.key
			CipherSuites:             ciphersuites,                    // ssl.cipher_suites
		},
	}

	return otelTLSConfig, nil
}
