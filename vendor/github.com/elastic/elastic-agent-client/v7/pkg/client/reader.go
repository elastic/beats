// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package client

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"

	protobuf "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// NewFromReader creates a new client reading the connection information from the io.Reader.
func NewFromReader(reader io.Reader, impl StateInterface, actions ...Action) (Client, error) {
	connInfo := &proto.ConnInfo{}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	err = protobuf.Unmarshal(data, connInfo)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(connInfo.PeerCert, connInfo.PeerKey)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(connInfo.CaCert)
	trans := credentials.NewTLS(&tls.Config{
		ServerName:   connInfo.ServerName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	})
	return New(connInfo.Addr, connInfo.Token, impl, actions, grpc.WithTransportCredentials(trans)), nil
}
