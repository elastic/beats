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

//go:build linux

package kprobes

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/auditbeat/tracing"
)

type perfChannel interface {
	C() <-chan any
	ErrC() <-chan error
	LostC() <-chan uint64
	Run() error
	Close() error
}

// probeChannel wraps a PerfChannel and takes ownership of the kprobes it
// registers in tracefs. Its Close method deregisters the probes after closing
// perf event fds, which is the order the kernel requires: a kprobe cannot be
// removed while a perf event still holds a reference to it (EBUSY).
type probeChannel struct {
	*tracing.PerfChannel
	tfs       *tracing.TraceFS
	installed []tracing.Probe

	once sync.Once
	err  error
}

var _ perfChannel = (*probeChannel)(nil)

// Close closes the perf event fds and then removes the registered kprobes.
// The sync.Once makes repeated and concurrent calls safe: PerfChannel.Close
// panics on a second call (it closes an already-closed channel internally).
//
// Close is idempotent and safe for concurrent use.
func (c *probeChannel) Close() error {
	c.once.Do(func() {
		// Uninstall even if PerfChannel.Close failed, so cleanup is
		// always attempted regardless of perf errors.
		c.err = errors.Join(c.PerfChannel.Close(), c.uninstall())
	})
	return c.err
}

func (c *probeChannel) uninstall() error {
	var errs []error
	for _, p := range c.installed {
		if err := c.tfs.RemoveKProbe(p); err != nil {
			errs = append(errs, err)
		}
	}
	c.installed = nil
	return errors.Join(errs...)
}

// newPerfChannel creates a probeChannel for the given probes. If any step
// fails, probes that were already registered are removed and the channel is
// closed before returning, leaving the kernel in a clean state.
func newPerfChannel(probes map[tracing.Probe]tracing.AllocateFn, ringSizeExponent int, bufferSize int, pid int) (_ *probeChannel, retErr error) {
	tfs, err := tracing.NewTraceFS()
	if err != nil {
		return nil, fmt.Errorf("error creating traceFS handler: %w", err)
	}

	pChannel, err := tracing.NewPerfChannel(
		tracing.WithTimestamp(),
		tracing.WithRingSizeExponent(ringSizeExponent),
		tracing.WithBufferSize(bufferSize),
		tracing.WithTID(pid),
		tracing.WithPollTimeout(200*time.Millisecond),
		tracing.WithWakeUpEvents(500),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating perf channel: %w", err)
	}

	pc := &probeChannel{
		PerfChannel: pChannel,
		tfs:         tfs,
	}

	defer func() {
		if retErr != nil {
			// Close removes registered probes after closing fds. The
			// rollback error is not joined to retErr: the original
			// cause is what the caller needs to see.
			_ = pc.Close()
		}
	}()

	// List existing probes before touching anything. If a probe from our
	// group is already present, remove it explicitly; treat failure (e.g.
	// EBUSY from a live concurrent instance) as fatal and surface it rather
	// than converting it to a confusing EEXIST from AddKProbe.
	existing, err := tfs.ListKProbes()
	if err != nil {
		return nil, fmt.Errorf("error listing existing kprobes: %w", err)
	}
	existingSet := make(map[[2]string]struct{}, len(existing))
	for _, p := range existing {
		existingSet[[2]string{p.Group, p.Name}] = struct{}{}
	}

	for probe, allocFn := range probes {
		if _, ok := existingSet[[2]string{probe.Group, probe.Name}]; ok {
			if err := tfs.RemoveKProbe(probe); err != nil {
				return nil, fmt.Errorf("error removing stale %s probe: %w", probe.Name, err)
			}
		}

		if err := tfs.AddKProbe(probe); err != nil {
			return nil, fmt.Errorf("error adding %s probe: %w", probe.Name, err)
		}
		// Record the probe immediately after a successful add, before any
		// fallible step, so the defer rollback can remove it on failure.
		pc.installed = append(pc.installed, probe)

		desc, err := tfs.LoadProbeFormat(probe)
		if err != nil {
			return nil, fmt.Errorf("error loading %s probe format data: %w", probe.Name, err)
		}

		decoder, err := tracing.NewStructDecoder(desc, allocFn)
		if err != nil {
			return nil, fmt.Errorf("error creating decoder for %s: %w", probe.Name, err)
		}

		if err := pChannel.MonitorProbe(desc, decoder); err != nil {
			return nil, fmt.Errorf("error monitoring %s probe: %w", probe.Name, err)
		}
	}

	return pc, nil
}
