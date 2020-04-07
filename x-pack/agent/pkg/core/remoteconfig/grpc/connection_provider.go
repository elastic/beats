// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package grpc

import (
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/remoteconfig"
)

var _ remoteconfig.ConnectionProvider = (*ConnectionProvider)(nil)
var _ grpcConnectionProvider = (*ConnectionProvider)(nil)

// ConnectionProvider is a connection provider for grpc connections
type ConnectionProvider struct {
	address          string
	caCrt            []byte
	clientPrivateKey []byte
	clientCert       []byte
}

type grpcConnectionProvider interface {
	remoteconfig.ConnectionProvider
	CA() []byte
	Cert() []byte
	PK() []byte
	IsSecured() bool
}

// NewConnectionProvider creates a new connection provider for grpc connections
func NewConnectionProvider(address string, caCrt []byte, clientPrivateKey, clientCert []byte) *ConnectionProvider {
	return &ConnectionProvider{
		address:          address,
		caCrt:            caCrt,
		clientPrivateKey: clientPrivateKey,
		clientCert:       clientCert,
	}
}

// Address returns an address used for connecting to a client
func (c *ConnectionProvider) Address() string { return c.address }

// CA returns a certificate authority associated with a connection
func (c *ConnectionProvider) CA() []byte { return c.caCrt }

// Cert returns a public certificate associated with a connection
func (c *ConnectionProvider) Cert() []byte { return c.clientCert }

// PK returns a private key associated with a connection
func (c *ConnectionProvider) PK() []byte { return c.clientPrivateKey }

// IsSecured returns true if all bits for setting up a secure connection were provided
func (c *ConnectionProvider) IsSecured() bool {
	return c.caCrt != nil && c.clientCert != nil && c.clientPrivateKey != nil
}
