// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package socket

import (
	"context"
	"net"
)

// DialContext returns a function that can be used to dial a local unix-domain socket.
func DialContext(socket string) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		d.LocalAddr = nil
		addr := net.UnixAddr{Name: socket, Net: "unix"}
		return d.DialContext(ctx, "unix", addr.String())
	}
}
