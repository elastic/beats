// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// isBrokenPipeOrEOFError checks for broken pipe, diconnect or EOF errors from osquery
// This is to workaround the known and possibly future defects with osquery implementation, allowing us to gracefully recover, restart osquery and rerun failed queries
func isBrokenPipeOrEOFError(err error) bool {
	var netErr *net.OpError
	return (errors.As(err, &netErr) && (errors.Is(netErr.Err, syscall.EPIPE) || errors.Is(netErr.Err, syscall.ECONNRESET))) ||
		strings.HasSuffix(err.Error(), " broken pipe") || strings.HasSuffix(err.Error(), " EOF")
}
