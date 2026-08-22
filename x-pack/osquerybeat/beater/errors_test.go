// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
)

func TestIsRecoverableOsqueryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "extension ping timeout",
			err:  errors.New("extension ping failed: timeout after 200ms"),
			want: true,
		},
		{
			name: "extension ping socket write timeout",
			err:  errors.New("extension ping failed: write unix @->/var/run/1980071180/osquery.sock: i/o timeout"),
			want: true,
		},
		{
			name: "wrapped extension ping timeout",
			err:  fmt.Errorf("run osquery: %w", errors.New("extension ping failed: timeout after 200ms")),
			want: true,
		},
		{
			name: "eof",
			err:  errors.New("osquery failed: EOF"),
			want: true,
		},
		{
			name: "broken pipe",
			err:  errors.New("write: broken pipe"),
			want: true,
		},
		{
			name: "epipe op error",
			err:  &net.OpError{Op: "write", Err: syscall.EPIPE},
			want: true,
		},
		{
			name: "net timeout",
			err:  &net.DNSError{Err: "i/o timeout", IsTimeout: true},
			want: true,
		},
		{
			name: "unrelated error is not recoverable",
			err:  errors.New("invalid configuration"),
			want: false,
		},
		{
			name: "non-timeout net error is not recoverable",
			err:  &net.DNSError{Err: "no such host"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRecoverableOsqueryError(tc.err); got != tc.want {
				t.Errorf("isRecoverableOsqueryError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
