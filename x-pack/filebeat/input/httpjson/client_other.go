// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package httpjson

import (
	"context"
	"errors"
	"net"
)

// npipeDialer implements transport.Dialer.
type npipeDialer struct {
	path string
}

func (npipeDialer) Dial(_, _ string) (net.Conn, error) {
	return nil, errors.New("named pipe only available on windows")
}

func (npipeDialer) DialContext(_ context.Context, _, _ string) (net.Conn, error) {
	return nil, errors.New("named pipe only available on windows")
}
