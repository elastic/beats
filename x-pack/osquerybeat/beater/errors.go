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

// extensionPingFailed is the prefix osquery-go uses when the extension manager
// watchdog fails to ping osqueryd. See osquery-go server.go.
const extensionPingFailed = "extension ping failed"

// isBrokenPipeOrEOFError checks for broken pipe, diconnect or EOF errors from osquery
// This is to workaround the known and possibly future defects with osquery implementation, allowing us to gracefully recover, restart osquery and rerun failed queries
func isBrokenPipeOrEOFError(err error) bool {
	var netErr *net.OpError
	return (errors.As(err, &netErr) && (errors.Is(netErr.Err, syscall.EPIPE) || errors.Is(netErr.Err, syscall.ECONNRESET))) ||
		strings.HasSuffix(err.Error(), " broken pipe") || strings.HasSuffix(err.Error(), " EOF")
}

// isRecoverableOsqueryError reports whether an osquery run failure should restart osqueryd
// rather than terminate the beat.
//
// This matters more when osquerybeat runs as a beat receiver than it did as a supervised
// process. A terminal error from Run is reported as status.Failed, which maps to an OTel
// PermanentError, and nothing restarts a receiver. The osquery_manager unit then stays
// FAILED until the whole agent is restarted.
func isRecoverableOsqueryError(err error) bool {
	if err == nil {
		return false
	}
	if isBrokenPipeOrEOFError(err) {
		return true
	}
	// osquery-go's extension manager watchdog shares a single thrift socket, and a single
	// 200ms lock, with extension registration and deregistration. Losing that lock race
	// surfaces as "extension ping failed: timeout after 200ms" and is indistinguishable
	// from osqueryd actually being gone. Either way, restarting osqueryd is the right move.
	if strings.Contains(err.Error(), extensionPingFailed) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
