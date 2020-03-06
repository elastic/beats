// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"google.golang.org/grpc/credentials"

	rpc "google.golang.org/grpc"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/remoteconfig"
)

// NewConnFactory creates a factory used to create connection. Hides implementation details
// of the underlying connections.
func NewConnFactory(backoffDelay, backoffMaxDelay time.Duration) remoteconfig.ConnectionCreator {
	return &connectionFactory{
		backoffDelay:    backoffDelay,
		backoffMaxDelay: backoffMaxDelay,
	}
}

type connectionFactory struct {
	backoffDelay    time.Duration
	backoffMaxDelay time.Duration
}

// NewConnection creates a connection
func (c *connectionFactory) NewConnection(provider remoteconfig.ConnectionProvider) (remoteconfig.Client, error) {
	if provider == nil {
		return nil, ErrProviderNotProvided
	}

	grpcProvider, ok := provider.(grpcConnectionProvider)
	if !ok {
		return nil, ErrProviderIncorrectType
	}

	if !grpcProvider.IsSecured() {
		conn, err := rpc.Dial(provider.Address(), rpc.WithInsecure())
		if err != nil {
			return nil, err
		}

		return CreateConfiguratorClient(conn, c.backoffDelay, c.backoffMaxDelay)
	}

	// Load client certificate
	pair, err := tls.X509KeyPair(grpcProvider.Cert(), grpcProvider.PK())
	if err != nil {
		return nil, errors.New(err, "creating client certificate pair")
	}

	// Load Cert Auth
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(grpcProvider.CA()); !ok {
		return nil, errors.New("failed to append client certificate to CA pool")
	}

	// Construct credentials
	creds := credentials.NewTLS(&tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{pair},
		ServerName:   "localhost",
	})

	conn, err := rpc.Dial(provider.Address(), rpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return CreateConfiguratorClient(conn, c.backoffDelay, c.backoffMaxDelay)
}
