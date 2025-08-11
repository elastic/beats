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
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/v7/auditbeat/tracing"
)

type perfChannel interface {
	C() <-chan interface{}
	ErrC() <-chan error
	LostC() <-chan uint64
	Run() error
	Close() error
}

func newPerfChannel(probes map[tracing.Probe]tracing.AllocateFn, ringSizeExponent int, bufferSize int, pid int) (*tracing.PerfChannel, error) {
	tfs, err := tracing.NewTraceFS()
	if err != nil {
		return nil, fmt.Errorf("error creating new tracefs: %w", err)
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
		return nil, fmt.Errorf("error creating new perf channel: %w", err)
	}

	for probe, allocFn := range probes {
		err = tfs.RemoveKProbe(probe)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error removing probe %s: %s\n", probe.Name, err)
		}

		fmt.Fprintf(os.Stdout, "Adding probe: %s at address %s\n", probe.Name, probe.Address)
		err := tfs.AddKProbe(probe)
		if err != nil {
			return nil, fmt.Errorf("error adding probe: %w", err)
		}
		desc, err := tfs.LoadProbeFormat(probe)
		if err != nil {
			return nil, fmt.Errorf("error loading probe format file for %s: %w", probe.Name, err)
		}

		decoder, err := tracing.NewStructDecoder(desc, allocFn)
		if err != nil {
			return nil, fmt.Errorf("error creating struct decoder for %s (%s): %w", probe.Name, probe.Address, err)
		}

		if err := pChannel.MonitorProbe(desc, decoder); err != nil {
			return nil, fmt.Errorf("error monitoring probe: %w", err)
		}
	}

	return pChannel, nil
}
