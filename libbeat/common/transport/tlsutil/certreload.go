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

package tlsutil

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// SetupCertReload configures TLS certificate hot-reload on the given
// tls.Config if the ServerConfig has certificate reload enabled. It creates
// a CertReloader that periodically re-reads cert/key files from disk and
// sets it as the GetCertificate callback, replacing the static Certificates
// slice.
//
// If tlsConfig or serverConfig is nil, or certificate reload is not enabled,
// this is a no-op.
func SetupCertReload(tlsConfig *tls.Config, serverConfig *tlscommon.ServerConfig) error {
	if tlsConfig == nil || serverConfig == nil {
		return nil
	}
	if !serverConfig.CertificateReload.IsEnabled() {
		return nil
	}

	certPath := serverConfig.Certificate.Certificate
	keyPath := serverConfig.Certificate.Key
	if certPath == "" || keyPath == "" {
		return nil
	}

	var opts []tlscommon.CertReloaderOption
	if serverConfig.CertificateReload.ReloadInterval > 0 {
		opts = append(opts, tlscommon.WithReloadInterval(serverConfig.CertificateReload.ReloadInterval))
	}

	passphrase, err := resolvePassphrase(serverConfig.Certificate)
	if err != nil {
		return fmt.Errorf("failed to resolve TLS key passphrase: %w", err)
	}
	if passphrase != "" {
		opts = append(opts, tlscommon.WithPassphrase(passphrase))
	}

	reloader, err := tlscommon.NewCertReloader(certPath, keyPath, opts...)
	if err != nil {
		return fmt.Errorf("failed to create certificate reloader: %w", err)
	}

	tlsConfig.GetCertificate = reloader.GetCertificate
	tlsConfig.Certificates = nil
	return nil
}

func resolvePassphrase(cert tlscommon.CertificateConfig) (string, error) {
	if cert.Passphrase != "" {
		return cert.Passphrase, nil
	}
	if cert.PassphrasePath != "" {
		data, err := os.ReadFile(cert.PassphrasePath)
		if err != nil {
			return "", fmt.Errorf("failed to read passphrase file %q: %w", cert.PassphrasePath, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}
