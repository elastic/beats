// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	rpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/remoteconfig/grpc"
)

const (
	serverAddressKey = "SERVER_ADDRESS"
)

// NewGrpcServer creates a server and pairs it with fleet.
// Reads secrets from provided reader, registers provided server
// and starts listening on negotiated address
func NewGrpcServer(secretsReader io.Reader, configServer grpc.ConfiguratorServer) error {
	// get creds from agent
	var cred *process.Creds
	secrets, err := ioutil.ReadAll(secretsReader)
	if err != nil {
		return errors.New(err, "failed to retrieve secrets from provided input")
	}

	err = yaml.Unmarshal(secrets, &cred)
	if err != nil {
		return errors.New(err, "failed to parse secrets from provided input")
	}

	// setup grpc server
	serverAddress, found := os.LookupEnv(serverAddressKey)
	if !found {
		return errors.New("server address not specified")
	}

	pair, err := tls.X509KeyPair(cred.Cert, cred.PK)
	if err != nil {
		return errors.New(err, "failed to load x509 key-pair")
	}

	// Create CA cert pool
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(cred.CaCert); !ok {
		errors.New("failed to append client certs")
	}

	fmt.Printf("Listening at %s\n", serverAddress)
	lis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("failed to start server: %v", serverAddress),
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, serverAddress))
	}

	// Create the TLS credentials
	serverCreds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{pair},
		ClientCAs:    certPool,
	})

	// Create the gRPC server with the credentials
	srv := rpc.NewServer(rpc.Creds(serverCreds))

	// Register the handler object
	grpc.RegisterConfiguratorServer(srv, configServer)

	// Serve and Listen
	if err := srv.Serve(lis); err != nil {
		return errors.New(err,
			fmt.Sprintf("grpc serve error: %s", serverAddress),
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, serverAddress))
	}

	return nil
}
