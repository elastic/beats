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

package processors

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type SharedProcessor struct {
	proc     beat.Processor
	cfg      uint64
	refCount int
}

var _ beat.Processor = (*SharedProcessor)(nil)

var sharedProcessorMu sync.Mutex
var sharedProcessors map[uint64]beat.Processor = make(map[uint64]beat.Processor)

// LoadOrStoreProcessor returns a shared instance of Processors for the given config, or creates a new one if it doesn't exist.
func LoadOrStoreProcessor(logger *logp.Logger, config *config.C, constructor Constructor) (beat.Processor, error) {
	sharedProcessorMu.Lock()
	defer sharedProcessorMu.Unlock()
	hash, err := cfgfile.HashConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to hash processor config: %w", err)
	}
	if p, ok := sharedProcessors[hash]; ok {
		if sharedProcessor, ok := p.(*SharedProcessor); ok {
			sharedProcessor.refCount++
			return sharedProcessor, nil
		}
		return nil, fmt.Errorf("unexpected non-shared processor found for hash %d", hash)
	}

	proc, err := constructor(config, logger)
	if err != nil {
		return nil, err
	}
	sharedProcessors[hash] = &SharedProcessor{
		proc:     proc,
		cfg:      hash,
		refCount: 1,
	}
	return sharedProcessors[hash], nil
}

func (p *SharedProcessor) Run(event *beat.Event) (*beat.Event, error) {
	return p.proc.Run(event)
}

func (p *SharedProcessor) String() string {
	return p.proc.String()
}

func (p *SharedProcessor) Close() error {
	sharedProcessorMu.Lock()
	defer sharedProcessorMu.Unlock()
	p.refCount--
	if p.refCount == 0 {
		delete(sharedProcessors, p.cfg)
		return Close(p.proc)
	}
	return nil
}
