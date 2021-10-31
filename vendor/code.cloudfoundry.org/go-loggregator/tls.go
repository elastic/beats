package loggregator

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
)

// NewIngressTLSConfig provides a convenient means for creating a *tls.Config
// which uses the CA, cert, and key for the ingress endpoint.
func NewIngressTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	return newTLSConfig(caPath, certPath, keyPath, "metron")
}

// NewEgressTLSConfig provides a convenient means for creating a *tls.Config
// which uses the CA, cert, and key for the egress endpoint.
func NewEgressTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	return newTLSConfig(caPath, certPath, keyPath, "reverselogproxy")
}

func newTLSConfig(caPath, certPath, keyPath, cn string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		ServerName:         cn,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}

	caCertBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, errors.New("cannot parse ca cert")
	}

	tlsConfig.RootCAs = caCertPool

	return tlsConfig, nil
}
