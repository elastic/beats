// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package client

import (
	"context"
	"net"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"

	"google.golang.org/grpc"
)

func dialContext(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, strings.TrimPrefix(control.Address(), "unix://"), grpc.WithInsecure(), grpc.WithContextDialer(dialer))
}

func dialer(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", addr)
}
