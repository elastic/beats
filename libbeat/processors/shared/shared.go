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

package shared

import (
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type sharedProcessorWithClose struct {
	beat.Processor
	hash     uint64
	refCount int

	sharedProcessors  map[uint64]*sharedProcessorWithClose
	sharedProcessorMu *sync.Mutex
}

var _ processors.PdataProcessor = (*sharedPdataProcessorWithClose)(nil)

// sharedPdataProcessorWithClose extends sharedProcessorWithClose with the
// pdata fast path. It is only created by New when the inner processor
// implements PdataProcessor.
type sharedPdataProcessorWithClose struct {
	*sharedProcessorWithClose
	pdataProc processors.PdataProcessor
}

func (p *sharedPdataProcessorWithClose) RunPdata(body pcommon.Map) (bool, error) {
	return p.pdataProc.RunPdata(body)
}

// New wraps a processor constructor to return a shared processor.
// The shared processor will be shared across all processors with the same configuration.
// The shared processor will be closed when the last processor using it is closed.
// Warning: the processor is built only once and then reused. Subsequent calls ignore the provided logger.
// Warning: To be used in conjunction with SafeProcessor. Ref: https://github.com/elastic/beats/blob/5586a1dcc31a748de8805e68c07094d08291fd7c/libbeat/processors/safe_processor.go#L133
func New(constructor processors.Constructor) processors.Constructor {
	sharedProcessors := make(map[uint64]*sharedProcessorWithClose)
	sharedProcessorMu := &sync.Mutex{}

	return func(cfg *config.C, logger *logp.Logger) (beat.Processor, error) {
		hash := uint64(0)
		if cfg != nil {
			var err error
			hash, err = cfgfile.HashConfig(cfg)
			if err != nil {
				return nil, err
			}
		}

		sharedProcessorMu.Lock()
		defer sharedProcessorMu.Unlock()
		if p, ok := sharedProcessors[hash]; ok {
			p.refCount++
			if pp, ok := p.Processor.(processors.PdataProcessor); ok {
				return &sharedPdataProcessorWithClose{p, pp}, nil
			}
			return p, nil
		}

		proc, err := constructor(cfg, logger)
		if err != nil {
			return nil, err
		}
		// if the processor does not implement `Closer` it does not need a wrap.
		// We can extend this in future, if needed.
		if _, ok := proc.(processors.Closer); !ok {
			return proc, nil
		}

		sw := &sharedProcessorWithClose{Processor: proc, hash: hash, sharedProcessors: sharedProcessors, sharedProcessorMu: sharedProcessorMu, refCount: 1}
		sharedProcessors[hash] = sw
		if pp, ok := proc.(processors.PdataProcessor); ok {
			return &sharedPdataProcessorWithClose{sw, pp}, nil
		}
		return sw, nil
	}
}

func (p *sharedProcessorWithClose) Close() error {
	p.sharedProcessorMu.Lock()
	defer p.sharedProcessorMu.Unlock()
	p.refCount--
	if p.refCount == 0 {
		delete(p.sharedProcessors, p.hash)
		return processors.Close(p.Processor)
	}
	return nil
}
