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

package processors_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// churnProcessor reports Run-after-Close and double-Close lifecycle violations.
type churnProcessor struct {
	closed atomic.Bool
	runs   atomic.Int64
	closes atomic.Int64
}

func (p *churnProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if p.closed.Load() {
		return nil, errors.New("BUG: Run called on underlying processor after Close")
	}
	p.runs.Add(1)
	return event, nil
}

func (p *churnProcessor) Close() error {
	p.closes.Add(1)
	if p.closed.Swap(true) {
		return errors.New("BUG: underlying processor closed twice")
	}
	return nil
}

func (p *churnProcessor) String() string { return "churn-processor" }

// TestSharedProcessorConcurrentChurn stresses sharing and isolation across configurations.
func TestSharedProcessorConcurrentChurn(t *testing.T) {
	const (
		procName       = "test-shared-churn-processor"
		churnWorkers   = 20
		churnIters     = 200
		runsPerIter    = 5
		steadyWorkers  = 30
		runsPerHold    = 20
		steadyReCycles = 50
	)

	var (
		createdMu sync.Mutex
		created   []*churnProcessor
	)
	wrapped := processors.SafeWrap(procName, func(*config.C, *logp.Logger) (beat.Processor, error) {
		p := &churnProcessor{}
		createdMu.Lock()
		defer createdMu.Unlock()
		created = append(created, p)
		return p, nil
	})

	cfgChurn := config.MustNewConfigFrom(map[string]any{"id": "churn"})
	cfgSteady := config.MustNewConfigFrom(map[string]any{"id": "steady"})

	logger := logptest.NewTestingLogger(t, "")
	event := &beat.Event{}

	var wg sync.WaitGroup

	// Churn workers: repeatedly construct, run, close the same config.
	for w := range churnWorkers {
		wg.Go(func() {
			for i := range churnIters {
				p, err := wrapped(cfgChurn, logger)
				if !assert.NoErrorf(t, err, "churn worker %d iter %d: construct failed", w, i) {
					return
				}
				for r := range runsPerIter {
					out, err := p.Run(event)
					assert.NoErrorf(t, err,
						"churn worker %d iter %d run %d: unexpected Run error on live ref", w, i, r)
					assert.Samef(t, event, out,
						"churn worker %d iter %d run %d: event not passed through", w, i, r)
				}
				assert.NoErrorf(t, processors.Close(p), "churn worker %d iter %d: Close failed", w, i)
			}
		})
	}

	// Steady workers: use a different config of the same processor name.
	for w := range steadyWorkers {
		wg.Go(func() {
			for c := range steadyReCycles {
				p, err := wrapped(cfgSteady, logger)
				if !assert.NoErrorf(t, err, "steady worker %d cycle %d: construct failed", w, c) {
					return
				}
				for r := range runsPerHold {
					out, err := p.Run(event)
					assert.NoErrorf(t, err,
						"steady worker %d cycle %d run %d: unexpected Run error on live ref", w, c, r)
					assert.Samef(t, event, out,
						"steady worker %d cycle %d run %d: event not passed through", w, c, r)
				}
				assert.NoErrorf(t, processors.Close(p), "steady worker %d cycle %d: Close failed", w, c)
			}
		})
	}

	wg.Wait()

	var totalRuns int64
	for i, p := range created {
		assert.Truef(t, p.closed.Load(), "underlying processor %d must be closed after all refs released", i)
		assert.EqualValuesf(t, 1, p.closes.Load(), "underlying processor %d must be closed exactly once", i)
		totalRuns += p.runs.Load()
	}
	expectedRuns := int64(churnWorkers*churnIters*runsPerIter + steadyWorkers*steadyReCycles*runsPerHold)
	assert.Equal(t, expectedRuns, totalRuns, "every Run must have been processed exactly once")

	before := len(created)
	p, err := wrapped(cfgChurn, logger)
	require.NoError(t, err)
	assert.Len(t, created, before+1, "a fresh underlying instance must be constructed after full release")
	_, err = p.Run(event)
	assert.NoError(t, err, "fresh instance must be runnable")
	assert.NoError(t, processors.Close(p))
}
