// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beater

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/fileset"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TestRunClosesRunReadyOnPreInitFailure guards against the regression in
// commit 00068f7 where the defer fb.runReady.Close() was placed after the
// PreInit call, leaving runReady unclosed when PreInit fails.  An unclosed
// runReady causes StopWithContext to wait the full five-second timeout.
func TestRunClosesRunReadyOnPreInitFailure(t *testing.T) {
	preInitErr := errors.New("preinit failed")
	fb := &Filebeat{
		done:           make(chan struct{}),
		runReady:       &closeOnce{ch: make(chan struct{})},
		logger:         logp.NewNopLogger(),
		moduleRegistry: new(fileset.ModuleRegistry),
	}
	b := &beat.Beat{
		Info:       beat.Info{Logger: logp.NewNopLogger()},
		Manager:    &preinitFailManager{err: preInitErr},
		Monitoring: beatmonitoring.NewMonitoring(),
	}

	err := fb.Run(b)
	require.ErrorIs(t, err, preInitErr)

	select {
	case <-fb.runReady.ch:
		// runReady was closed — StopWithContext will not block
	default:
		t.Fatal("runReady was not closed after PreInit failure; StopWithContext would wait five seconds")
	}
}

// preinitFailManager is a management.Manager stub whose PreInit returns an
// error.  Only the methods called by Run before PreInit are implemented; any
// other call panics, catching unexpected code paths during the test.
type preinitFailManager struct {
	management.Manager // zero-valued; panics on any unimplemented method
	err                error
}

func (m *preinitFailManager) Enabled() bool { return true }
func (m *preinitFailManager) PreInit() error { return m.err }
func (m *preinitFailManager) RegisterDiagnosticHook(_, _, _, _ string, _ management.DiagnosticHook) {
}

// TestStopWaitsForRunReady proves that Stop does not close the done channel
// until Run has closed runReady (i.e., reached waitFinished.Wait).
// This guards against a race where the OTel collector calls Shutdown before
// the beat's Run goroutine has initialised its shutdown-signal machinery.
func TestStopWaitsForRunReady(t *testing.T) {
	fb := &Filebeat{
		done:     make(chan struct{}),
		runReady: &closeOnce{ch: make(chan struct{})},
		logger:   logp.NewNopLogger(),
	}

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		fb.Stop()
	}()

	// done must still be open: Stop is waiting for runReady to be closed.
	select {
	case <-fb.done:
		t.Fatal("Stop closed done before Run closed runReady")
	default:
	}

	// Simulate Run() reaching the waitFinished.Wait() call.
	fb.runReady.Close()

	select {
	case <-fb.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop did not close done after runReady was closed")
	}

	<-stopDone
}
