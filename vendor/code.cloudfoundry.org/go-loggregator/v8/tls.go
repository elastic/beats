package loggregator

import (
	"crypto/tls"

	"code.cloudfoundry.org/tlsconfig"
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
	return tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(certPath, keyPath),
	).Client(
		tlsconfig.WithAuthorityFromFile(caPath),
		tlsconfig.WithServerName(cn),
	)
}
