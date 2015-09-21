package outputs

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"

	"github.com/elastic/libbeat/logp"
)

var (
	// ErrNotACertificate indicates a PEM file to be loaded not being a valid
	// PEM file or certificate.
	ErrNotACertificate = errors.New("file is not a certificate")

	// ErrCertificateNoKey indicate a configuration error with missing key file
	ErrCertificateNoKey = errors.New("key file not configured")

	// ErrKeyNoCertificate indicate a configuration error with missing certificate file
	ErrKeyNoCertificate = errors.New("certificate file not configured")
)

// LoadTLSConfig will load a certificate from config with all TLS based keys
// defined. If Certificate and CertificateKey are configured, client authentication
// will be configured. If no CAs are configured, the host CA will be used by go
// built-in TLS support.
func LoadTLSConfig(config MothershipConfig) (*tls.Config, error) {
	certificate := config.Certificate
	key := config.CertificateKey
	rootCAs := config.CAs
	hasCertificate := certificate != ""
	hasKey := key != ""

	var certs []tls.Certificate
	switch {
	case hasCertificate && !hasKey:
		return nil, ErrCertificateNoKey
	case !hasCertificate && hasKey:
		return nil, ErrKeyNoCertificate
	case hasCertificate && hasKey:
		cert, err := tls.LoadX509KeyPair(certificate, key)
		if err != nil {
			logp.Critical("Failed loading client certificate", err)
			return nil, err
		}
		certs = []tls.Certificate{cert}
	}

	var roots *x509.CertPool
	if len(rootCAs) > 0 {
		roots = x509.NewCertPool()
		for _, caFile := range rootCAs {
			pemData, err := ioutil.ReadFile(caFile)
			if err != nil {
				logp.Critical("Failed reading CA certificate: %s", err)
				return nil, err
			}

			if ok := roots.AppendCertsFromPEM(pemData); !ok {
				return nil, ErrNotACertificate
			}
		}
	}

	// Support minimal TLS 1.0.
	// TODO: check supported JRuby versions for logstash supported
	//       TLS 1.1 and switch
	tlsConfig := tls.Config{
		MinVersion:   tls.VersionTLS10,
		Certificates: certs,
		RootCAs:      roots,
	}
	return &tlsConfig, nil
}
