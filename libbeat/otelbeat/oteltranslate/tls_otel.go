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
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/config/configtls"

	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var tlsVersions = map[uint16]string{
	tls.VersionTLS10: "1.0",
	tls.VersionTLS11: "1.1",
	tls.VersionTLS12: "1.2",
	tls.VersionTLS13: "1.3",
}

func validateUnsupportedConfig(output *config.C) error {

	if sslConfig, err := output.Child("ssl", -1); err == nil {
		if sslConfig.HasField("curve_type") {
			return fmt.Errorf("ssl.curve_types is currently not supported: %w", errors.ErrUnsupported)
		} else if sslConfig.HasField("renegotiation") {
			return fmt.Errorf("ssl.renegotiation is currently not supported: %w", errors.ErrUnsupported)
		}

		if reloadCfg, err := sslConfig.Child("restart_on_cert_change", -1); err == nil {
			if reloadCfg.HasField("enabled") {
				return fmt.Errorf("ssl.restart_on_cert_change.enabled is currently not supported: %w", errors.ErrUnsupported)
			} else if reloadCfg.HasField("period") {
				return fmt.Errorf("ssl.restart_on_cert_change.period is currently not supported: %w", errors.ErrUnsupported)
			}
		}

	}

	return nil
}

// TLSCommonToOTel converts a tlscommon.Config into the OTel configtls.ClientConfig
// ca_trusted_fingerprint, ca_sha_256 and verification mode should be handled separately by beatsauth extension
func TLSCommonToOTel(output *config.C, logger *logp.Logger) (map[string]any, error) {
	logger = logger.Named("tls-to-otel")
	otelTLSConfig := map[string]any{}

	var tlsCfg = elasticsearch.ESDefaultTransportSettings()

	err := output.Unpack(&tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed unpacking config: %w", err)
	}

	if tlsCfg.TLS == nil {
		return nil, nil
	}

	if !tlsCfg.TLS.IsEnabled() {
		return map[string]any{
			"insecure": true,
		}, nil
	}

	// throw error if unsupported tls config is set
	if err := validateUnsupportedConfig(output); err != nil {
		return nil, err
	}

	tlscfg := tlsCfg.TLS
	// validate the beats config before proceeding
	if err := tlscfg.Validate(); err != nil {
		return nil, err
	}

	// handle verification_mode:none
	// Handle all other cases, including VerifyFull, VerifyCertificate, or VerifyStrict by beatsauth extension
	if tlscfg.VerificationMode == tlscommon.VerifyNone {
		otelTLSConfig["insecure_skip_verify"] = true
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

	tlsConfig, err := tlscommon.LoadTLSConfig(tlscfg, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot load SSL configuration: %w", err)
	}
	goTLSConfig := tlsConfig.ToConfig()

	// unpacks -> ssl.cipher_suites
	ciphersuites := []string{}
	for _, cs := range goTLSConfig.CipherSuites {
		ciphersuites = append(ciphersuites, tls.CipherSuiteName(cs))
	}

	setIfNotNil(otelTLSConfig, "include_system_ca_certs_pool", includeSystemCACertsPool)
	setIfNotNil(otelTLSConfig, "ca_pem", strings.Join(caCerts, "")) // ssl.certificate_authorities
	setIfNotNil(otelTLSConfig, "cert_pem", certPem)                 // ssl.certificate
	setIfNotNil(otelTLSConfig, "key_pem", certKeyPem)               // ssl.key
	setIfNotNil(otelTLSConfig, "cipher_suites", ciphersuites)       // ssl.cipher_suites

	otelTLSConfig["min_version"] = tlsVersions[goTLSConfig.MinVersion]
	otelTLSConfig["max_version"] = tlsVersions[goTLSConfig.MaxVersion]

	if err := typeSafetyCheck(otelTLSConfig); err != nil {
		return nil, err
	}
	return otelTLSConfig, nil
}

// For type safety check
func typeSafetyCheck(value map[string]any) error {
	// the returned valued should match `clienttls.Config` type.
	// it throws an error if non existing key names  are set
	var result configtls.ClientConfig
	d, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Squash:      true,
		Result:      &result,
		ErrorUnused: true,
	})

	err := d.Decode(value)
	if err != nil {
		return err
	}
	return err
}

// Helper function to conditionally add fields to the map
func setIfNotNil(m map[string]any, key string, value any) {
	if value == nil {
		return
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		if v.String() != "" {
			m[key] = value
		}
	case reflect.Bool:
		if v.Bool() {
			m[key] = value
		}
	case reflect.Map, reflect.Slice:
		if v.Len() > 0 {
			m[key] = value
		}
	default:
		m[key] = value
	}
}
