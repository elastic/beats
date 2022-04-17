// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common/transport/tlscommon"
)

// NewProviderFromCertificate returns a TokenProvider that uses certificate-based
// authentication.
func NewProviderFromCertificate(
	endpoint, resource, applicationID, tenantID string,
	conf tlscommon.CertificateConfig) (sptp TokenProvider, err error) {
	cert, privKey, err := loadConfigCerts(conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed loading certificates")
	}
	oauth, err := adal.NewOAuthConfig(endpoint, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "error generating OAuthConfig")
	}

	spt, err := adal.NewServicePrincipalTokenFromCertificate(
		*oauth,
		applicationID,
		cert,
		privKey,
		resource,
	)
	if err != nil {
		return nil, err
	}
	spt.SetAutoRefresh(true)
	return (*servicePrincipalToken)(spt), nil
}

func loadConfigCerts(cfg tlscommon.CertificateConfig) (cert *x509.Certificate, key *rsa.PrivateKey, err error) {
	tlsCert, err := tlscommon.LoadCertificate(&cfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error loading X509 certificate from '%s'", cfg.Certificate)
	}
	if tlsCert == nil || len(tlsCert.Certificate) == 0 {
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
