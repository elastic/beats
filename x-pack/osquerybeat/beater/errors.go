// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"errors"
	"net"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
)

// isBrokenPipeOrEOFError checks for broken pipe, diconnect or EOF errors from osquery
// This is to workaround the known and possibly future defects with osquery implementation, allowing us to gracefully recover, restart osquery and rerun failed queries
func isBrokenPipeOrEOFError(err error) bool {
	var netErr *net.OpError
	return (errors.As(err, &netErr) && (errors.Is(netErr.Err, syscall.EPIPE) || errors.Is(netErr.Err, syscall.ECONNRESET))) ||
		strings.HasSuffix(err.Error(), " broken pipe") || strings.HasSuffix(err.Error(), " EOF")
}
<<<<<<< HEAD
=======

// extensionPingFailed is the prefix osquery-go adds when the extension manager
// watchdog loses communication with osqueryd.
// https://github.com/osquery/osquery-go/blob/0cc22f415e57/server.go#L274
const extensionPingFailed = "extension ping failed"

// isRecoverableOsqueryError reports whether an osquery run failure should restart osqueryd
// rather than terminate the beat.
//
// This matters more when osquerybeat runs as a beat receiver than it did as a supervised
// process. A terminal error from Run is reported as status.Failed, which maps to an OTel
// PermanentError, and nothing restarts a receiver. The osquery_manager unit then stays
// FAILED until the whole agent is restarted.
func isRecoverableOsqueryError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, errOsqueryRuntimeExit78) || isBrokenPipeOrEOFError(err) {
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

// isOsqueryExit78 recognizes the same exit used for both startup configuration
// errors and catastrophic runtime logger failures.
func isOsqueryExit78(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var exitError *exec.ExitError
	return errors.As(err, &exitError) && exitError.ExitCode() == 78
}

// osqueryRunState distinguishes runtime failure from invalid startup configuration.
// Exit seals the state before logger cleanup, so late config callbacks cannot
// turn a startup failure into a recoverable runtime failure.
type osqueryRunState struct {
	state atomic.Uint32
	// Written by the process-exit callback; read only after all run goroutines finish.
	startupErr error
}

const (
	osqueryStarting uint32 = iota
	osqueryReady
	osqueryExited
)

var errOsqueryRuntimeExit78 = errors.New("osqueryd runtime exit 78")

func (s *osqueryRunState) ready() {
	s.state.CompareAndSwap(osqueryStarting, osqueryReady)
}

func (s *osqueryRunState) exit(err error) error {
	ready := s.state.Swap(osqueryExited) == osqueryReady
	if isOsqueryExit78(err) {
		if ready {
			return errors.Join(errOsqueryRuntimeExit78, err)
		}
		s.startupErr = err
	}
	return err
}

// run classifies the process error before Run waits for output logging and cleanup.
func (s *osqueryRunState) run(ctx context.Context, runner osqd.Runner, flags osqd.Flags) error {
	var exitErr error
	err := runner.Run(ctx, flags, func(err error) { exitErr = s.exit(err) })
	if err != nil && exitErr != nil {
		return exitErr
	}
	return err
}

// result must be called after the run's errgroup finishes. A startup exit 78
// takes precedence over an extension error reported during process cleanup.
func (s *osqueryRunState) result(parent context.Context, err error) error {
	if s.startupErr != nil {
		if parent.Err() != nil {
			return parent.Err()
		}
		return s.startupErr
	}
	return err
}
>>>>>>> c37abd0 ([osquerybeat] Recover from runtime osqueryd exit 78 (#53776))
