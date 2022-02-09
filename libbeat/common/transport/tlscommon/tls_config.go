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
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// TLSConfig is the interface used to configure a tcp client or server from a `Config`
type TLSConfig struct {

	// List of allowed SSL/TLS protocol versions. Connections might be dropped
	// after handshake succeeded, if TLS version in use is not listed.
	Versions []TLSVersion

	// Configure SSL/TLS verification mode used during handshake. By default
	// VerifyFull will be used.
	Verification TLSVerificationMode

	// List of certificate chains to present to the other side of the
	// connection.
	Certificates []tls.Certificate

	// Set of root certificate authorities use to verify server certificates.
	// If RootCAs is nil, TLS might use the system its root CA set (not supported
	// on MS Windows).
	RootCAs *x509.CertPool

	// Set of root certificate authorities use to verify client certificates.
	// If ClientCAs is nil, TLS might use the system its root CA set (not supported
	// on MS Windows).
	ClientCAs *x509.CertPool

	// List of supported cipher suites. If nil, a default list provided by the
	// implementation will be used.
	CipherSuites []CipherSuite

	// Types of elliptic curves that will be used in an ECDHE handshake. If empty,
	// the implementation will choose a default.
	CurvePreferences []tls.CurveID

	// Renegotiation controls what types of renegotiation are supported.
	// The default, never, is correct for the vast majority of applications.
	Renegotiation tls.RenegotiationSupport

	// ClientAuth controls how we want to verify certificate from a client, `none`, `optional` and
	// `required`, default to required. Do not affect TCP client.
	ClientAuth tls.ClientAuthType

	// CASha256 is the CA certificate pin, this is used to validate the CA that will be used to trust
	// the server certificate.
	CASha256 []string

	// ServerName is the remote server we're connecting to. It can be a hostname or IP address.
	ServerName string

	// time returns the current time as the number of seconds since the epoch.
	// If time is nil, TLS uses time.Now.
	time func() time.Time
}

var (
	MissingPeerCertificate = errors.New("missing peer certificates")
)

// ToConfig generates a tls.Config object. Note, you must use BuildModuleClientConfig to generate a config with
// ServerName set, use that method for servers with SNI.
// By default VerifyConnection is set to client mode.
func (c *TLSConfig) ToConfig() *tls.Config {
	if c == nil {
		return &tls.Config{}
	}

	minVersion, maxVersion := extractMinMaxVersion(c.Versions)

	insecure := c.Verification != VerifyStrict
	if c.Verification == VerifyNone {
		logp.NewLogger("tls").Warn("SSL/TLS verifications disabled.")
	}

	return &tls.Config{
		MinVersion:         minVersion,
		MaxVersion:         maxVersion,
		Certificates:       c.Certificates,
		RootCAs:            c.RootCAs,
		ClientCAs:          c.ClientCAs,
		InsecureSkipVerify: insecure,
		CipherSuites:       convCipherSuites(c.CipherSuites),
		CurvePreferences:   c.CurvePreferences,
		Renegotiation:      c.Renegotiation,
		ClientAuth:         c.ClientAuth,
		Time:               c.time,
		VerifyConnection:   makeVerifyConnection(c),
	}
}

// BuildModuleConfig takes the TLSConfig and transform it into a `tls.Config`.
func (c *TLSConfig) BuildModuleClientConfig(host string) *tls.Config {
	if c == nil {
		// use default TLS settings, if config is empty.
		return &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true,
			VerifyConnection: makeVerifyConnection(&TLSConfig{
				Verification: VerifyFull,
				ServerName:   host,
			}),
		}
	}

	// Make a copy of c, because we're gonna mutate it after
	// calling ToConfig. ToConfig calls a function that creates
	// a closure that needs to access cc. A shallow copy is enough
	// because all slice/pointer fields won't be modified.
	cc := *c

	// Keep a copy of the host (wheather an IP or hostname)
	// for later validation. It is used by makeVerifyConnection
	cc.ServerName = host
	config := cc.ToConfig()

	// config.ServerName does not verify IP addresses
	config.ServerName = host

	return config
}

// BuildServerConfig takes the TLSConfig and transform it into a `tls.Config` for server side objects.
func (c *TLSConfig) BuildServerConfig(host string) *tls.Config {
	if c == nil {
		// use default TLS settings, if config is empty.
		return &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true,
			VerifyConnection: makeVerifyServerConnection(&TLSConfig{
				Verification: VerifyFull,
				ServerName:   host,
			}),
		}
	}

	config := c.ToConfig()
	config.ServerName = host
	config.VerifyConnection = makeVerifyServerConnection(c)
	return config
}

func makeVerifyConnection(cfg *TLSConfig) func(tls.ConnectionState) error {
	switch cfg.Verification {
	case VerifyFull:
		// Cert is trusted by CA
		// Hostname or IP matches the certificate
		// tls.Config.InsecureSkipVerify  is set to true
		return func(cs tls.ConnectionState) error {
			// On the client side, PeerCertificates can't be empty.
			if len(cs.PeerCertificates) == 0 {
				return MissingPeerCertificate
			}

			opts := x509.VerifyOptions{
				Roots:         cfg.RootCAs,
				Intermediates: x509.NewCertPool(),
			}
			if err := verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts); err != nil {
				return err
			}
			return verifyHostname(cs.PeerCertificates[0], cfg.ServerName)
		}
	case VerifyCertificate:
		// Cert is trusted by CA
		// Does NOT validate hostname or IP addresses
		// tls.Config.InsecureSkipVerify  is set to true
		return func(cs tls.ConnectionState) error {
			// On the client side, PeerCertificates can't be empty.
			if len(cs.PeerCertificates) == 0 {
				return MissingPeerCertificate
			}

			opts := x509.VerifyOptions{
				Roots:         cfg.RootCAs,
				Intermediates: x509.NewCertPool(),
			}
			return verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
		}
	case VerifyStrict:
		// Cert is trusted by CA
		// Hostname or IP matches the certificate
		// Returns error if SNA is empty
		// The whole validation is done by Go's standard library default
		// SSL/TLS verification (tls.Config.InsecureSkipVerify is set to false)
		// so we only need to check the pin
		if len(cfg.CASha256) > 0 {
			return func(cs tls.ConnectionState) error {
				return verifyCAPin(cfg.CASha256, cs.VerifiedChains)
			}
		}
	default:
	}

	return nil

}

func makeVerifyServerConnection(cfg *TLSConfig) func(tls.ConnectionState) error {
	switch cfg.Verification {
	case VerifyFull:
		return func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
					return MissingPeerCertificate
				}
				return nil
			}

			opts := x509.VerifyOptions{
				Roots:         cfg.ClientCAs,
				Intermediates: x509.NewCertPool(),
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			}
			err := verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
			if err != nil {
				return err
			}
			return verifyHostname(cs.PeerCertificates[0], cs.ServerName)
		}
	case VerifyCertificate:
		return func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
					return MissingPeerCertificate
				}
				return nil
			}

			opts := x509.VerifyOptions{
				Roots:         cfg.ClientCAs,
				Intermediates: x509.NewCertPool(),
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			}
			return verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
		}
	case VerifyStrict:
		if len(cfg.CASha256) > 0 {
			return func(cs tls.ConnectionState) error {
				return verifyCAPin(cfg.CASha256, cs.VerifiedChains)
			}
		}
	default:
	}

	return nil

}

func verifyCertsWithOpts(certs []*x509.Certificate, casha256 []string, opts x509.VerifyOptions) error {
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	verifiedChains, err := certs[0].Verify(opts)
	if err != nil {
		return err
	}

	if len(casha256) > 0 {
		return verifyCAPin(casha256, verifiedChains)
	}
	return nil
}

// verifyHostname verifies if the provided hostnmae matches
// cert.DNSNames, cert.OPAddress (SNA)
// For hostnames, if SNA is empty, validate the hostname against cert.Subject.CommonName
func verifyHostname(cert *x509.Certificate, hostname string) error {
	if hostname == "" {
		return nil
	}
	// check if the server name is an IP
	ip := hostname
	if len(ip) >= 3 && ip[0] == '[' && ip[len(ip)-1] == ']' {
		ip = ip[1 : len(ip)-1]
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		for _, certIP := range cert.IPAddresses {
			if parsedIP.Equal(certIP) {
				return nil
			}
		}

		parsedCNIP := net.ParseIP(cert.Subject.CommonName)
		if parsedCNIP != nil {
			if parsedIP.Equal(parsedCNIP) {
				return nil
			}
		}

		return x509.HostnameError{Certificate: cert, Host: hostname}
	}

	dnsnames := cert.DNSNames
	if len(dnsnames) == 0 || len(dnsnames) == 1 && dnsnames[0] == "" {
		if cert.Subject.CommonName != "" {
			dnsnames = []string{cert.Subject.CommonName}
		}
	}

	for _, name := range dnsnames {
		if matchHostnames(name, hostname) {
			if !validHostname(name, true) {
				return fmt.Errorf("invalid hostname in cert")
			}
			return nil
		}
	}
	return x509.HostnameError{Certificate: cert, Host: hostname}
}
