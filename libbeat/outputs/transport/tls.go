package transport

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

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

	// List of supported cipher suites. If nil, a default list provided by the
	// implementation will be used.
	CipherSuites []uint16

	// Types of elliptic curves that will be used in an ECDHE handshake. If empty,
	// the implementation will choose a default.
	CurvePreferences []tls.CurveID
}

type TLSVersion uint16

const (
	TLSVersionSSL30 TLSVersion = tls.VersionSSL30
	TLSVersion10    TLSVersion = tls.VersionTLS10
	TLSVersion11    TLSVersion = tls.VersionTLS11
	TLSVersion12    TLSVersion = tls.VersionTLS12
)

type TLSVerificationMode uint8

const (
	VerifyFull TLSVerificationMode = iota
	VerifyNone

	// TODO: add VerifyCertificate support. Due to checks being run
	//       during handshake being limited, verify certificates in
	//       postVerifyTLSConnection
	// VerifyCertificate
)

var tlsDefaultVersions = []TLSVersion{
	TLSVersion10,
	TLSVersion11,
	TLSVersion12,
}

func TLSDialer(
	forward Dialer,
	config *TLSConfig,
	timeout time.Duration,
) (Dialer, error) {
	var lastTLSConfig *tls.Config
	var lastNetwork string
	var lastAddress string
	var m sync.Mutex

	return DialerFunc(func(network, address string) (net.Conn, error) {
		switch network {
		case "tcp", "tcp4", "tcp6":
		default:
			return nil, fmt.Errorf("unsupported network type %v", network)
		}

		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		var tlsConfig *tls.Config
		m.Lock()
		if network == lastNetwork && address == lastAddress {
			tlsConfig = lastTLSConfig
		}
		if tlsConfig == nil {
			tlsConfig = config.BuildModuleConfig(host)
			lastNetwork = network
			lastAddress = address
			lastTLSConfig = tlsConfig
		}
		m.Unlock()

		return tlsDialWith(forward, network, address, timeout, tlsConfig, config)
	}), nil
}

func tlsDialWith(
	dialer Dialer,
	network, address string,
	timeout time.Duration,
	tlsConfig *tls.Config,
	config *TLSConfig,
) (net.Conn, error) {
	socket, err := dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}

	conn := tls.Client(socket, tlsConfig)

	withTimeout := timeout > 0
	if withTimeout {
		if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}

	if err := conn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// remove timeout if handshake was subject to timeout:
	if withTimeout {
		conn.SetDeadline(time.Time{})
	}

	if err := postVerifyTLSConnection(conn, config); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func postVerifyTLSConnection(conn *tls.Conn, config *TLSConfig) error {
	st := conn.ConnectionState()

	if !st.HandshakeComplete {
		return errors.New("incomplete handshake")
	}

	// no more checks if no extra configs available
	if config == nil {
		return nil
	}

	versions := config.Versions
	if versions == nil {
		versions = tlsDefaultVersions
	}
	versionOK := false
	for _, version := range versions {
		versionOK = versionOK || st.Version == uint16(version)
	}
	if !versionOK {
		return fmt.Errorf("tls version %v not configured", TLSVersion(st.Version))
	}

	return nil
}

func (c *TLSConfig) BuildModuleConfig(host string) *tls.Config {
	if c == nil {
		// use default TLS settings, if config is empty.
		return &tls.Config{ServerName: host}
	}

	versions := c.Versions
	if len(versions) == 0 {
		versions = tlsDefaultVersions
	}

	minVersion := uint16(0xffff)
	maxVersion := uint16(0)
	for _, version := range versions {
		v := uint16(version)
		if v < minVersion {
			minVersion = v
		}
		if v > maxVersion {
			maxVersion = v
		}
	}

	insecure := c.Verification != VerifyFull
	if insecure {
		logp.Warn("SSL/TLS verifications disabled.")
	}

	return &tls.Config{
		ServerName:         host,
		MinVersion:         minVersion,
		MaxVersion:         maxVersion,
		Certificates:       c.Certificates,
		RootCAs:            c.RootCAs,
		InsecureSkipVerify: insecure,
		CipherSuites:       c.CipherSuites,
		CurvePreferences:   c.CurvePreferences,
	}
}

var tlsProtocolVersions = map[string]TLSVersion{
	"SSLv3":   TLSVersionSSL30,
	"SSLv3.0": TLSVersionSSL30,
	"TLSv1":   TLSVersion10,
	"TLSv1.0": TLSVersion10,
	"TLSv1.1": TLSVersion11,
	"TLSv1.2": TLSVersion12,
}

func (v TLSVersion) String() string {
	versions := map[TLSVersion]string{
		TLSVersionSSL30: "SSLv3",
		TLSVersion10:    "TLSv1.0",
		TLSVersion11:    "TLSv1.1",
		TLSVersion12:    "TLSv1.2",
	}
	if s, ok := versions[v]; ok {
		return s
	}
	return "unknown"
}

func (v *TLSVersion) Unpack(in interface{}) error {
	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("tls version must be an identifier")
	}

	version, found := tlsProtocolVersions[s]
	if !found {
		return fmt.Errorf("invalid tls version '%v'", s)
	}

	*v = version
	return nil
}

var tlsVerificationModes = map[string]TLSVerificationMode{
	"":     VerifyFull,
	"full": VerifyFull,
	"none": VerifyNone,
	// "certificate": verifyCertificate,
}

func (m TLSVerificationMode) String() string {
	modes := map[TLSVerificationMode]string{
		VerifyFull: "full",
		// VerifyCertificate: "certificate",
		VerifyNone: "none",
	}

	if s, ok := modes[m]; ok {
		return s
	}
	return "unknown"
}

func (m *TLSVerificationMode) Unpack(in interface{}) error {
	if in == nil {
		*m = VerifyFull
		return nil
	}

	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("verification mode must be an identifier")
	}

	mode, found := tlsVerificationModes[s]
	if !found {
		return fmt.Errorf("unknown verification mode '%v'", s)
	}

	*m = mode
	return nil
}
