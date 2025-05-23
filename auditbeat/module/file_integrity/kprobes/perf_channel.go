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
		return nil, err
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
		return nil, err
	}

	for probe, allocFn := range probes {
		_ = tfs.RemoveKProbe(probe)

		err := tfs.AddKProbe(probe)
		if err != nil {
			return nil, err
		}
		desc, err := tfs.LoadProbeFormat(probe)
		if err != nil {
			return nil, err
		}

		decoder, err := tracing.NewStructDecoder(desc, allocFn)
		if err != nil {
			return nil, err
		}

		if err := pChannel.MonitorProbe(desc, decoder); err != nil {
			return nil, err
		}
	}

	return pChannel, nil
}
