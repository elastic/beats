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
	"errors"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var ErrClosed = errors.New("attempt to use a closed processor")

type SafeProcessor struct {
	beat.Processor
	closed uint32
}

// Run allows to run processor only when `Close` was not called prior
func (p *SafeProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		return nil, ErrClosed
	}
	return p.Processor.Run(event)
}

// Close makes sure the underlying `Close` function is called only once.
func (p *SafeProcessor) Close() (err error) {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return Close(p.Processor)
	}
	logp.L().Warnf("tried to close already closed %q processor", p.Processor.String())
	return nil
}

// SafeWrap makes sure that the processor handles all the required edge-cases.
//
// Each processor might end up in multiple processor groups.
// Every group has its own `Close` that calls `Close` on each
// processor of that group which leads to multiple `Close` calls
// on the same processor.
//
// If `SafeWrap` is not used, the processor must handle multiple
// `Close` calls by using `sync.Once` in its `Close` function.
// We make it easer for processor developers and take care of it
// in the processor registry instead.
func SafeWrap(constructor Constructor) Constructor {
	return func(config *config.C) (beat.Processor, error) {
		processor, err := constructor(config)
		if err != nil {
			return nil, err
		}
		// if the processor does not implement `Closer`
		// it does not need a wrap
		if _, ok := processor.(Closer); !ok {
			return processor, nil
		}

		return &SafeProcessor{
			Processor: processor,
		}, nil
	}
}
