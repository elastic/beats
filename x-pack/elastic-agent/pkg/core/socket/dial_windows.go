// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package socket

import (
	"context"
	"net"

	winio "github.com/Microsoft/go-winio"
)

// DialContext returns a function that can be used to dial a local Windows npipe.
func DialContext(socket string) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		return winio.DialPipeContext(ctx, `\\.\pipe\`+socket)
	}
}
