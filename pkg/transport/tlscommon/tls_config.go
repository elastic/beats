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
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// certPoolProvider abstracts over static and dynamically-reloaded CA pools.
// CAReloader implements this interface for the dynamic case; staticCertPool
// implements it for the static case.
type certPoolProvider interface {
	GetCertPool() *x509.CertPool
	AddTrustedCert(cert *x509.Certificate)
	IsDynamic() bool
}

// staticCertPool is a certPoolProvider backed by a fixed *x509.CertPool.
// It is safe for concurrent use.
type staticCertPool struct {
	mu   sync.Mutex
	pool *x509.CertPool
}

func newStaticCertPool(pool *x509.CertPool) *staticCertPool {
	return &staticCertPool{pool: pool}
}

func (s *staticCertPool) GetCertPool() *x509.CertPool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pool
}

func (s *staticCertPool) AddTrustedCert(cert *x509.Certificate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pool == nil {
		s.pool = x509.NewCertPool()
	}
	s.pool.AddCert(cert)
}

func (s *staticCertPool) IsDynamic() bool { return false }

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

	// rootCAs provides the CA pool for verifying server certificates.
	// nil means use the system pool. Use currentRootCAs() to access.
	rootCAs certPoolProvider

	// clientCAs provides the CA pool for verifying client certificates.
	// nil means use the system pool. Use currentClientCAs() to access.
	clientCAs certPoolProvider

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

	// CATrustedFingerprint is the HEX encoded fingerprint of a CA certificate. If present in the chain
	// this certificate will be added to the list of trusted CAs (RootCAs) during the handshake.
	CATrustedFingerprint string

	// ServerName is the remote server we're connecting to. It can be a hostname or IP address.
	ServerName string

	// time returns the current time as the number of seconds since the epoch.
	// If time is nil, TLS uses time.Now.
	time func() time.Time

	Logger *logp.Logger

	// certReloader, when set, provides dynamic certificate reloading from disk.
	// ToConfig will use it to set GetCertificate and GetClientCertificate on
	// the resulting tls.Config instead of populating Certificates statically.
	certReloader *CertReloader
}

var (
	ErrMissingPeerCertificate = errors.New("missing peer certificates")
)

func (c *TLSConfig) currentRootCAs() *x509.CertPool {
	if c.rootCAs == nil {
		return nil
	}
	return c.rootCAs.GetCertPool()
}

func (c *TLSConfig) currentClientCAs() *x509.CertPool {
	if c.clientCAs == nil {
		return nil
	}
	return c.clientCAs.GetCertPool()
}

type tlsOptFunc func(t *TLSSettings)

func (t tlsOptFunc) apply(c *TLSSettings) {
	t(c)
}

type TLSOption interface {
	apply(t *TLSSettings)
}

type TLSSettings struct {
	logger *logp.Logger
}

func WithLogger(logger *logp.Logger) TLSOption {
	return tlsOptFunc(func(t *TLSSettings) {
		t.logger = logger
	})
}

// ToConfig generates a tls.Config object. Note, you must use BuildModuleClientConfig to generate a config with
// ServerName set, use that method for servers with SNI.
// By default VerifyConnection is set to client mode.
func (c *TLSConfig) ToConfig() *tls.Config {
	if c == nil {
		return &tls.Config{}
	}

	minVersion, maxVersion := extractMinMaxVersion(c.Versions)

	dynamic := (c.rootCAs != nil && c.rootCAs.IsDynamic()) || (c.clientCAs != nil && c.clientCAs.IsDynamic())
	insecure := c.Verification != VerifyStrict || dynamic
	if c.Verification == VerifyNone {
		c.Logger.Named("tls").Warn("SSL/TLS verifications disabled.")
	}

	cfg := &tls.Config{
		MinVersion:         minVersion,
		MaxVersion:         maxVersion,
		Certificates:       c.Certificates,
		RootCAs:            c.currentRootCAs(),
		ClientCAs:          c.currentClientCAs(),
		InsecureSkipVerify: insecure, //nolint:gosec // we are using our own verification for now
		CipherSuites:       convCipherSuites(c.CipherSuites),
		CurvePreferences:   c.CurvePreferences,
		Renegotiation:      c.Renegotiation,
		ClientAuth:         c.ClientAuth,
		Time:               c.time,
		VerifyConnection:   makeVerifyConnection(c, c.Logger),
	}

	if c.certReloader != nil {
		cfg.GetCertificate = c.certReloader.GetCertificate
		cfg.GetClientCertificate = c.certReloader.GetClientCertificate
	}

	return cfg
}

// BuildModuleClientConfig takes the TLSConfig and transform it into a `tls.Config`.
func (c *TLSConfig) BuildModuleClientConfig(host string, options ...TLSOption) *tls.Config {
	var settings TLSSettings
	for _, opt := range options {
		opt.apply(&settings)
	}

	if settings.logger == nil {
		settings.logger = logp.NewLogger("")
	}

	if c == nil {
		// use default TLS settings, if config is empty.
		return &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true, //nolint:gosec // we are using our own verification for now
			VerifyConnection: makeVerifyConnection(&TLSConfig{
				Verification: VerifyFull,
				ServerName:   host,
			}, settings.logger.Named("tls")),
		}
	}

	// Make a copy of c, because we're gonna mutate it after
	// calling ToConfig. ToConfig calls a function that creates
	// a closure that needs to access cc. A shallow copy is enough
	// because all slice/pointer fields won't be modified.
	cc := *c

	// Keep a copy of the host (whether an IP or hostname)
	// for later validation. It is used by makeVerifyConnection
	cc.ServerName = host
	config := cc.ToConfig()

	// config.ServerName does not verify IP addresses
	config.ServerName = host

	return config
}

// BuildServerConfig takes the TLSConfig and transform it into a `tls.Config`
// for server side connections.
func (c *TLSConfig) BuildServerConfig(host string) *tls.Config {
	if c == nil {
		// use default TLS settings, if config is empty.
		return &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true, //nolint:gosec // we are using our own verification for now
			VerifyConnection: makeVerifyServerConnection(&TLSConfig{
				Verification: VerifyCertificate,
				ServerName:   host,
			}),
		}
	}

	config := c.ToConfig()
	config.ServerName = host
	config.VerifyConnection = makeVerifyServerConnection(c)
	return config
}

func trustRootCA(cfg *TLSConfig, peerCerts []*x509.Certificate, logger *logp.Logger) error {
	logger = logger.Named("tls")
	logger.Debug("'ca_trusted_fingerprint' set, looking for matching fingerprints")
	fingerprint, err := hex.DecodeString(cfg.CATrustedFingerprint)
	if err != nil {
		return fmt.Errorf("decode 'ca_trusted_fingerprint': %w", err)
	}

	foundCADigests := []string{}

	for _, cert := range peerCerts {

		// Compute digest for each certificate.
		digest := sha256.Sum256(cert.Raw)

		if cert.IsCA {
			foundCADigests = append(foundCADigests, hex.EncodeToString(digest[:]))
		}

		if !bytes.Equal(digest[0:], fingerprint) {
			continue
		}

		// Make sure the fingerprint matches a CA certificate
		if !cert.IsCA {
			logger.Warn("Certificate matching 'ca_trusted_fingerprint' found, but it is not a CA certificate. 'ca_trusted_fingerprint' can only be used to trust CA certificates.")
			continue
		}

		logger.Debug("CA certificate matching 'ca_trusted_fingerprint' found, adding it to 'certificate_authorities'")
		cfg.rootCAs.AddTrustedCert(cert)
		return nil
	}

	// if we are here, we didn't find any CA certificate matching the fingerprint
	if len(foundCADigests) == 0 {
		logger.Warn("The remote server's certificate is presented without its certificate chain. Using 'ca_trusted_fingerprint' requires that the server presents a certificate chain that includes the certificate's issuing certificate authority.")
	} else {
		logger.Warnf("The provided 'ca_trusted_fingerprint': '%s' does not match the fingerprint of any Certificate Authority present in the server's certificate chain. Found the following CA fingerprints instead: %v", cfg.CATrustedFingerprint, foundCADigests)
	}

	return nil
}

func makeVerifyConnection(cfg *TLSConfig, logger *logp.Logger) func(tls.ConnectionState) error {
	serverName := cfg.ServerName

	switch cfg.Verification {
	case VerifyNone:
		// Chain and hostname verification are disabled, but FIPS key-type constraints
		// still apply — all peer certificates are checked for approved key types.
		return fipsVerifyNoneCallback()
	case VerifyFull:
		// Cert is trusted by CA
		// Hostname or IP matches the certificate
		// tls.Config.InsecureSkipVerify is set to true
		return func(cs tls.ConnectionState) error {
			if cfg.CATrustedFingerprint != "" {
				if err := trustRootCA(cfg, cs.PeerCertificates, logger); err != nil {
					return err
				}
			}
			// On the client side, PeerCertificates can't be empty.
			if len(cs.PeerCertificates) == 0 {
				return ErrMissingPeerCertificate
			}

			opts := x509.VerifyOptions{
				Roots:         cfg.currentRootCAs(),
				Intermediates: x509.NewCertPool(),
			}
			chains, err := verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
			if err != nil {
				return err
			}

			if err := verifyHostname(cs.PeerCertificates[0], serverName); err != nil {
				return err
			}

			return checkAllChainsFIPS(chains)
		}
	case VerifyCertificate:
		// Cert is trusted by CA
		// Does NOT validate hostname or IP addresses
		// tls.Config.InsecureSkipVerify is set to true
		return func(cs tls.ConnectionState) error {
			if cfg.CATrustedFingerprint != "" {
				if err := trustRootCA(cfg, cs.PeerCertificates, logger); err != nil {
					return err
				}
			}
			// On the client side, PeerCertificates can't be empty.
			if len(cs.PeerCertificates) == 0 {
				return ErrMissingPeerCertificate
			}

			opts := x509.VerifyOptions{
				Roots:         cfg.currentRootCAs(),
				Intermediates: x509.NewCertPool(),
			}
			chains, err := verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
			if err != nil {
				return err
			}
			return checkAllChainsFIPS(chains)
		}
	case VerifyStrict:
		// Cert is trusted by CA
		// Hostname or IP matches the certificate
		// Returns error if SAN is empty
		return func(cs tls.ConnectionState) error {
			if cfg.CATrustedFingerprint != "" {
				if err := trustRootCA(cfg, cs.PeerCertificates, logger); err != nil {
					return err
				}
			}
			// On the client side, PeerCertificates can't be empty.
			if len(cs.PeerCertificates) == 0 {
				return ErrMissingPeerCertificate
			}
			if cfg.rootCAs != nil && cfg.rootCAs.IsDynamic() {
				// When rootCAs is dynamic, InsecureSkipVerify is true so Go's
				// stdlib won't validate the chain. Do full strict verification
				// manually using the dynamically reloaded CA pool.
				opts := x509.VerifyOptions{
					Roots:         cfg.currentRootCAs(),
					DNSName:       serverName,
					Intermediates: x509.NewCertPool(),
				}
				chains, err := verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
				if err != nil {
					return err
				}
				return checkAllChainsFIPS(chains)
			}
			// Static CAs: Go's stdlib handles chain + hostname verification
			// (InsecureSkipVerify is false). We only need a callback for CA pin or FIPS checking.
			if len(cfg.CASha256) > 0 {
				if err := verifyCAPin(cfg.CASha256, cs.VerifiedChains); err != nil {
					return err
				}
				// CA pin verified — the verified chain is guaranteed non-empty.
				return checkAllChainsFIPS(cs.VerifiedChains)
			}
			return checkConnectionCertsFIPS(cs)
		}
	default:
		// Unrecognised modes are rejected at config validation time, so this
		// branch is unreachable in practice. An error callback rather than nil
		// ensures the handshake fails explicitly if it is ever reached.
		mode := cfg.Verification
		return func(_ tls.ConnectionState) error {
			return fmt.Errorf("tlscommon: unhandled TLSVerificationMode %v", mode)
		}
	}
}

func makeVerifyServerConnection(cfg *TLSConfig) func(tls.ConnectionState) error {
	switch cfg.Verification {

	case VerifyNone:
		// Chain and client-cert verification are disabled, but FIPS key-type
		// constraints still apply — all presented certificates are checked for
		// approved key types.
		return fipsVerifyNoneCallback()
	// VerifyFull would attempt to match 'host' (c.ServerName) that is the host
	// the client is trying to connect to with a DNS, IP or the CN from the
	// client's certificate. Such validation, besides making no sense on the
	// server side also causes errors as the client certificate usually does not
	// contain a DNS, IP or CN matching the server's hostname.
	case VerifyFull, VerifyCertificate:
		return verifyClientChain(cfg)
	case VerifyStrict:
		if cfg.clientCAs != nil && cfg.clientCAs.IsDynamic() {
			return verifyClientChain(cfg)
		}
		if len(cfg.CASha256) > 0 {
			return func(cs tls.ConnectionState) error {
				if len(cs.PeerCertificates) == 0 {
					return noPeerCertsError(cfg)
				}
				if err := verifyCAPin(cfg.CASha256, cs.VerifiedChains); err != nil {
					return err
				}
				// CA pin verified — the verified chain is guaranteed non-empty.
				return checkAllChainsFIPS(cs.VerifiedChains)
			}
		}
		// Static CAs: the TLS stack builds a verified chain for RequireAndVerifyClientCert
		// and VerifyClientCertIfGiven; FIPS checks use those chains.
		// For RequireAnyClientCert, chain building is deliberately skipped, so only
		// the certificates the client chose to send are checked. A client that omits
		// intermediates can pass with a non-FIPS intermediate — accepted limitation;
		// use RequireAndVerifyClientCert for full chain coverage.
		return func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return noPeerCertsError(cfg)
			}
			return checkConnectionCertsFIPS(cs)
		}
	default:
		// Unrecognised modes are rejected at config validation time, so this
		// branch is unreachable in practice. An error callback rather than nil
		// ensures the handshake fails explicitly if it is ever reached.
		mode := cfg.Verification
		return func(_ tls.ConnectionState) error {
			return fmt.Errorf("tlscommon: unhandled TLSVerificationMode %v", mode)
		}
	}
}

func verifyCertsWithOpts(certs []*x509.Certificate, casha256 []string, opts x509.VerifyOptions) ([][]*x509.Certificate, error) {
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	chains, err := certs[0].Verify(opts)
	if err != nil {
		return nil, err
	}
	if len(casha256) > 0 {
		return chains, verifyCAPin(casha256, chains)
	}
	return chains, nil
}

// verifyClientChain returns a callback that verifies the client certificate
// chain against the configured CA pool and enforces FIPS key-type constraints.
func verifyClientChain(cfg *TLSConfig) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return noPeerCertsError(cfg)
		}
		opts := x509.VerifyOptions{
			Roots:         cfg.currentClientCAs(),
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		}
		chains, err := verifyCertsWithOpts(cs.PeerCertificates, cfg.CASha256, opts)
		if err != nil {
			return err
		}
		return checkAllChainsFIPS(chains)
	}
}

// noPeerCertsError returns ErrMissingPeerCertificate when the config requires a
// verified client cert and none was presented, otherwise nil.
func noPeerCertsError(cfg *TLSConfig) error {
	if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
		return ErrMissingPeerCertificate
	}
	return nil
}

// checkConnectionCertsFIPS enforces FIPS key-type constraints using the verified
// chains when Go has built them, falling back to the raw peer certs when
// VerifiedChains is empty. Go only populates VerifiedChains for ClientAuth modes
// that fully verify the client cert (RequireAndVerifyClientCert,
// VerifyClientCertIfGiven); for RequireAnyClientCert, VerifiedChains is empty
// even when PeerCertificates is not.
func checkConnectionCertsFIPS(cs tls.ConnectionState) error {
	if len(cs.VerifiedChains) > 0 {
		return checkAllChainsFIPS(cs.VerifiedChains)
	}
	return checkPeerCertsFIPS(cs.PeerCertificates)
}

// verifyHostname verifies if the provided hostnmae matches
// cert.DNSNames, cert.IPAddress (SNA)
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
