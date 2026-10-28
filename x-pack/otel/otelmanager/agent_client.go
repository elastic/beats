// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/grpc"
	gproto "google.golang.org/protobuf/proto"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/api/npipe"
)

// NewAgentClient returns a client for the control protocol, connecting with
// the connection info from Elastic Agent as a beat process does.
// connectionSecret is the base64-encoded connection info, in the format beat
// processes read from stdin.
func NewAgentClient(connectionSecret string) (client.V2, error) {
	connInfo, err := base64.StdEncoding.DecodeString(connectionSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid management.agent.connection_secret: %w", err)
	}
	var info proto.StartUpInfo
	if err := gproto.Unmarshal(connInfo, &info); err != nil {
		return nil, fmt.Errorf("invalid management.agent.connection_secret: %w", err)
	}

	closing := &closingClient{conns: make(map[*grpc.ClientConn]struct{})}
	dialOpts := []grpc.DialOption{grpc.WithChainStreamInterceptor(closing.trackConn)}
	if u, err := url.Parse(info.Addr); err == nil && u.Scheme == "npipe" {
		// gRPC can't dial named pipes itself
		dialPipe := npipe.DialContext(`\\.\pipe\` + strings.TrimPrefix(u.Path, "/"))
		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return dialPipe(ctx, "", "")
		}))
	}

	versionInfo := client.VersionInfo{
		Name:      "beat-v2-client-for-receiver",
		BuildHash: version.Commit(),
		Meta: map[string]string{
			"commit":     version.Commit(),
			"build_time": version.BuildTime().String(),
		},
	}
	agentClient, _, err := client.NewV2FromReader(bytes.NewReader(connInfo), versionInfo, client.WithGRPCDialOptions(dialOpts...))
	if err != nil {
		return nil, fmt.Errorf("error creating Elastic Agent client: %w", err)
	}
	closing.V2 = agentClient
	return closing, nil
}

// closingClient closes the client's gRPC connections when it stops. The
// client dials a connection in Start but never closes it, which is fine for a
// beat process, but leaks the connection every time a receiver is recreated.
//
// TODO: remove once elastic-agent-client closes its connection in Stop.
type closingClient struct {
	client.V2

	mx    sync.Mutex
	conns map[*grpc.ClientConn]struct{}
}

// trackConn is a stream interceptor that records the client's connection. The
// client only uses streams, so this sees every connection it dials.
func (c *closingClient) trackConn(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	c.mx.Lock()
	c.conns[cc] = struct{}{}
	c.mx.Unlock()
	return streamer(ctx, desc, cc, method, opts...)
}

// Stop stops the client and closes its connections.
func (c *closingClient) Stop() {
	c.V2.Stop()
	c.mx.Lock()
	defer c.mx.Unlock()
	for cc := range c.conns {
		_ = cc.Close()
	}
	clear(c.conns)
}
