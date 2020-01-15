// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package grpc

import (
	context "context"
	"errors"
	"time"

	grpc "google.golang.org/grpc"
	rpc "google.golang.org/grpc"

	"github.com/elastic/beats/libbeat/common/backoff"
	"github.com/elastic/beats/x-pack/agent/pkg/core/remoteconfig"
)

var (
	// ErrNotGrpcClient is used when connection passed into a factory is not a grpc connection
	ErrNotGrpcClient = errors.New("not a grpc client")
	// ErrProviderNotProvided is used when provider passed into factory is not provided
	ErrProviderNotProvided = errors.New("provider not provided")
	// ErrProviderIncorrectType is used when provider passed into factory does not implement grpcConnectionProvided
	ErrProviderIncorrectType = errors.New("provided provider has incorrect type")
)

// CreateConfiguratorClient creates a new client from a connection passed in.
// This wraps generated grpc implementation so the change of the underlying
// technology is just the change of the namespace.
func CreateConfiguratorClient(conn interface{}, delay, maxDelay time.Duration) (remoteconfig.ConfiguratorClient, error) {
	grpcConn, ok := conn.(*rpc.ClientConn)
	if !ok {
		return nil, ErrNotGrpcClient
	}

	var boff backoff.Backoff
	done := make(chan struct{})

	if delay > 0 && maxDelay > 0 {
		boff = backoff.NewEqualJitterBackoff(done, delay, maxDelay)
	} else {
		// no retry strategy configured
		boff = NewNoopBackoff()
	}

	return &client{
		grpcConn: grpcConn,
		client:   NewConfiguratorClient(grpcConn),
		backoff:  boff,
		done:     done,
	}, nil
}

type client struct {
	grpcConn *grpc.ClientConn
	client   ConfiguratorClient
	backoff  backoff.Backoff
	done     chan struct{}
}

func (c *client) Config(ctx context.Context, config string) error {
	request := ConfigRequest{
		Config: string(config),
	}

	_, err := c.client.Config(ctx, &request)
	backoff.WaitOnError(c.backoff, err)

	return err
}

func (c *client) Close() error {
	close(c.done)
	return c.grpcConn.Close()
}

func (c *client) Backoff() backoff.Backoff {
	return c.backoff
}
