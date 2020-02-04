// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

type sptProviderFromCert struct {
	sync.Mutex
	certs         tlscommon.CertificateConfig
	applicationID string
	endpoint      string
	resource      string
	tenantID      string
	spt           *adal.ServicePrincipalToken
}

// NewProviderFromCertificate returns a TokenProvider that uses certificate-based
// authentication.
func NewProviderFromCertificate(
	endpoint, resource, applicationID, tenantID string,
	conf tlscommon.CertificateConfig) (sptp TokenProvider, err error) {
	provider := &sptProviderFromCert{
		certs:         conf,
		applicationID: applicationID,
		resource:      resource,
		endpoint:      endpoint,
		tenantID:      tenantID,
	}
	if provider.spt, err = provider.getServicePrincipalToken(tenantID); err != nil {
		return nil, err
	}
	provider.spt.SetAutoRefresh(true)
	return provider, nil
}

// Token returns an oauth token that can be used for bearer authorization.
func (provider *sptProviderFromCert) Token() (string, error) {
	provider.Mutex.Lock()
	defer provider.Mutex.Unlock()
	if err := provider.spt.EnsureFresh(); err != nil {
		return "", errors.Wrap(err, "refreshing spt token")
	}
	token := provider.spt.Token()
	return token.OAuthToken(), nil
}

// Renew re-authenticates with the oauth2 endpoint to get a new Service Principal Token.
func (provider *sptProviderFromCert) Renew() error {
	provider.Mutex.Lock()
	defer provider.Mutex.Unlock()
	return provider.spt.Refresh()
}

func (provider *sptProviderFromCert) getServicePrincipalToken(tenantID string) (*adal.ServicePrincipalToken, error) {
	cert, privKey, err := loadConfigCerts(provider.certs)
	if err != nil {
		return nil, errors.Wrap(err, "failed loading certificates")
	}
	oauth, err := adal.NewOAuthConfig(provider.endpoint, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "error generating OAuthConfig")
	}

	return adal.NewServicePrincipalTokenFromCertificate(
		*oauth,
		provider.applicationID,
		cert,
		privKey,
		provider.resource,
	)
}

func loadConfigCerts(cfg tlscommon.CertificateConfig) (cert *x509.Certificate, key *rsa.PrivateKey, err error) {
	tlsCert, err := tlscommon.LoadCertificate(&cfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error loading X509 certificate from '%s'", cfg.Certificate)
	}
	if len(tlsCert.Certificate) < 1 {
		return nil, nil, fmt.Errorf("no certificates loaded from '%s'", cfg.Certificate)
	}
	cert, err = x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error parsing X509 certificate from '%s'", cfg.Certificate)
	}
	if tlsCert.PrivateKey == nil {
		return nil, nil, fmt.Errorf("failed loading private key from '%s'", cfg.Key)
	}
	key, ok := tlsCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("private key at '%s' is not an RSA private key", cfg.Key)
	}
	return cert, key, nil
}
