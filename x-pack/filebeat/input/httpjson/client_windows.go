// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package httpjson

import (
	"context"
	"net"
	"path/filepath"

	"github.com/Microsoft/go-winio"
)

// npipeDialer implements transport.Dialer to a constant named pipe path.
type npipeDialer struct {
	path string
}

func (d npipeDialer) Dial(_, _ string) (net.Conn, error) {
	return winio.DialPipe(`\\.\pipe`+filepath.FromSlash(d.path), nil)
}

func (d npipeDialer) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, `\\.\pipe`+filepath.FromSlash(d.path))
}
