package tlsconfig

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

// PoolOption is an functional option type that can be used to configure a
// certificate pool.
type PoolOption func(*x509.CertPool) error

// PoolBuilder is used to build a certificate pool. You normally won't need to
// Build this yourself and instead should use the WithAuthorityBuilder and
// WithClientAuthenticationBuilder functions.
type PoolBuilder struct {
	base *x509.CertPool
	opts []PoolOption
	err  error
}

// Build creates the certificate pool.
func (pb PoolBuilder) Build() (*x509.CertPool, error) {
	if pb.err != nil {
		return nil, pb.err
	}

	for _, opt := range pb.opts {
		if err := opt(pb.base); err != nil {
			return nil, err
		}
	}

	return pb.base, nil
}

// FromEmptyPool creates a PoolBuilder from an empty certificate pool. The
// options passed can amend the returned pool.
func FromEmptyPool(opts ...PoolOption) PoolBuilder {
	return PoolBuilder{
		base: x509.NewCertPool(),
		opts: opts,
	}
}

// FromSystemPool creates a PoolBuilder from the system's certificate pool. The
// options passed can amend the returned pool.
func FromSystemPool(opts ...PoolOption) PoolBuilder {
	pool, err := x509.SystemCertPool()
	return PoolBuilder{
		base: pool,
		err:  err,
		opts: opts,
	}
}

// WithCertsFromFile will add all of the certificates found in a PEM-encoded
// file to a certificate pool.
func WithCertsFromFile(path string) PoolOption {
	return func(pool *x509.CertPool) error {
		pemCerts, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read certificate(s) at path %q: %s", path, err)
		}

		certsRead := 0
		for len(pemCerts) > 0 {
			var block *pem.Block
			block, pemCerts = pem.Decode(pemCerts)
			if block == nil {
				break
			}
			if len(block.Headers) != 0 {
				return fmt.Errorf("unexpected headers in PEM block in file %q: %v", path, block.Headers)
			}
			if block.Type != "CERTIFICATE" {
				return fmt.Errorf("unexpected PEM block type %q in file %q", block.Type, path)
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse certificate in %q: %s", path, err)
			}

			if err := WithCert(cert)(pool); err != nil {
				return fmt.Errorf("failed to add certificate in file %q to pool: %s", path, err)
			}

			certsRead++
		}

		if certsRead == 0 {
			return fmt.Errorf("no valid certificates read from file %q", path)
		}

		return nil
	}
}

// WithCert will add the certificate directly to a certificate pool.
func WithCert(cert *x509.Certificate) PoolOption {
	return func(pool *x509.CertPool) error {
		// We do not check if the certificate is valid here in case that a user
		// has an expired root certificate that they never use in their system
		// certificate store.
		//
		// Perhaps we can only check the expiration in the case the that
		// certificate is user specified?
		pool.AddCert(cert)
		return nil
	}
}
