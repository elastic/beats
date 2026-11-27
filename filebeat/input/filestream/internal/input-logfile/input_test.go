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

package input_logfile

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

// restartingProspector requests one restart from its first Run and raises
// its harvester limit in Restart. Every later Run blocks until the input is
// cancelled and stops its harvesters, like the filestream prospector.
type restartingProspector struct {
	noopProspector

	// cancelOnRestart cancels the input from the Run that requests the
	// restart.
	cancelOnRestart context.CancelFunc
	// restarted is closed when the second Run starts.
	restarted chan struct{}

	limit            uint64
	restartRequested bool
	takeOvers        int
	restarts         int
	runners          []*harvesterRunner
	limits           []uint64
}

func (p *restartingProspector) TakeOver(_ StoreUpdater, _ func(Source) string) error {
	p.takeOvers++
	return nil
}

func (p *restartingProspector) Run(ctx v2.Context, _ StateMetadataUpdater, hg HarvesterGroup, _ *Metrics) {
	runner, _ := hg.(*harvesterRunner)
	p.runners = append(p.runners, runner)
	p.limits = append(p.limits, runner.harvesterLimit)

	if len(p.runners) == 1 {
		p.restartRequested = true
		if p.cancelOnRestart != nil {
			p.cancelOnRestart()
		}
		return
	}
	if p.restarted != nil {
		close(p.restarted)
	}
	<-ctx.Cancelation.Done()
	_ = hg.StopHarvesters()
}

func (p *restartingProspector) HarvesterLimit() uint64 { return p.limit }

func (p *restartingProspector) RestartRequested() bool { return p.restartRequested }

func (p *restartingProspector) Restart() {
	p.restarts++
	p.restartRequested = false
	p.limit = 500
}

// newRestartingInput creates the input of p. Its Run returns once the
// prospector stopped for good.
func newRestartingInput(t *testing.T, p *restartingProspector) v2.Input {
	t.Helper()
	setupCacheForTest(t)

	manager := &InputManager{
		Logger:     logp.NewNopLogger(),
		StateStore: createSampleStore(t, nil).WithGCPeriod(time.Minute),
		Type:       "filestream",
		Configure: func(_ *config.C, _ *logp.Logger, _ *SourceIdentifier) (Prospector, Harvester, error) {
			return p, &mockHarvester{}, nil
		},
	}
	t.Cleanup(manager.Close)

	inp, err := manager.Create(config.MustNewConfigFrom(map[string]any{
		"id":              "restart-test",
		"harvester_limit": p.limit,
	}))
	require.NoError(t, err, "Create must succeed")
	return inp
}

func inputContext(t *testing.T, ctx context.Context) v2.Context {
	return v2.Context{
		Logger:          logptest.NewTestingLogger(t, ""),
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}
}

func TestManagedInputRestart(t *testing.T) {
	t.Run("runs the prospector again with a new harvester group", func(t *testing.T) {
		requireNoEngine(t)
		ctx, cancel := context.WithCancel(t.Context())
		p := &restartingProspector{limit: 3, restarted: make(chan struct{})}
		inp := newRestartingInput(t, p)

		errCh := make(chan error, 1)
		go func() {
			errCh <- inp.Run(inputContext(t, ctx), &MockPipeline{})
		}()
		select {
		case <-p.restarted:
		case err := <-errCh:
			require.FailNow(
				t,
				"Run returned before the restart",
				"error: %v",
				err,
			)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the prospector must be run again")
		}
		sharedEngine.Lock()
		refs := sharedEngine.refs
		sharedEngine.Unlock()
		assert.Equal(t, 1, refs, "the input must hold one engine reference")

		cancel()
		assert.NoError(t, <-errCh, "Run must succeed")

		assert.Equal(t, 1, p.takeOvers, "TakeOver must run once")
		assert.Equal(t, 1, p.restarts, "Restart must run once")
		assert.Equal(
			t,
			[]uint64{3, 500},
			p.limits,
			"the new harvester group must get the limit read after Restart",
		)
		require.Len(t, p.runners, 2, "Run must be called twice")
		assert.NotSame(
			t,
			p.runners[0],
			p.runners[1],
			"a restart must build a new harvester group",
		)
		assert.Same(
			t,
			p.runners[0].engine,
			p.runners[1].engine,
			"a restart must keep the engine",
		)
		assert.True(
			t,
			p.runners[0].closed,
			"the first harvester group must be stopped",
		)
		requireNoEngine(t)
	})

	t.Run("does not restart once cancelled", func(t *testing.T) {
		requireNoEngine(t)
		ctx, cancel := context.WithCancel(t.Context())
		p := &restartingProspector{limit: 3, cancelOnRestart: cancel}
		inp := newRestartingInput(t, p)

		assert.NoError(
			t,
			inp.Run(inputContext(t, ctx), &MockPipeline{}),
			"Run must succeed",
		)

		assert.Equal(
			t,
			0,
			p.restarts,
			"Restart must not run after cancellation",
		)
		require.Len(t, p.runners, 1, "Run must be called once")
		assert.True(
			t,
			p.runners[0].closed,
			"the harvester group must be stopped",
		)
		requireNoEngine(t)
	})
}
