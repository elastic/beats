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

package module

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// batchedRunner synchronizes all periodic metricsets in a module so they
// fetch in parallel on a single ticker, then each metricset's events are
// sent via client.PublishAll. This gives one ConsumeLogs call per metricset
// per cycle instead of one per event.
type batchedRunner struct {
	done      chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once

	msws       []*metricSetWrapper
	clients    []beat.Client // parallel to msws
	module     mb.Module
	monitoring beatmonitoring.Monitoring
	logger     *logp.Logger
}

func newBatchedRunner(
	clients []beat.Client,
	msws []*metricSetWrapper,
	module mb.Module,
	mon beatmonitoring.Monitoring,
	logger *logp.Logger,
) *batchedRunner {
	return &batchedRunner{
		done:       make(chan struct{}),
		msws:       msws,
		clients:    clients,
		module:     module,
		monitoring: mon,
		logger:     logger.Named("module"),
	}
}

func (br *batchedRunner) Start() {
	br.startOnce.Do(func() {
		br.logger.Debugf("Starting batched runner for %s with %d metricsets",
			br.module.Name(), len(br.msws))
		moduleList.Add(br.module.Name())

		// Register monitoring for each metricset.
		registry := br.monitoring.InputsRegistry()
		for _, msw := range br.msws {
			metricsPath := msw.ID()
			registry.Add(metricsPath, msw.Metrics(), monitoring.Full)
			monitoring.NewString(msw.Metrics(), "starttime").Set(common.Time(time.Now()).String())
			msw.periodic = true
		}

		br.module.UpdateStatus(status.Starting, fmt.Sprintf("batched runner for %s is starting", br.module.Name()))

		br.wg.Add(1)
		go br.run()
	})
}

func (br *batchedRunner) Stop() {
	br.stopOnce.Do(func() {
		close(br.done)
		br.wg.Wait()

		// Cleanup: close clients, deregister monitoring, close metricsets.
		for i, msw := range br.msws {
			br.clients[i].Close()
			br.monitoring.InputsRegistry().Remove(msw.ID())
			releaseStats(br.monitoring.StatsRegistry(), msw.stats)
			msw.close()
		}
		moduleList.Remove(br.module.Name())

		br.logger.Debugf("Stopped batched runner for %s", br.module.Name())
	})
}

func (br *batchedRunner) run() {
	defer br.wg.Done()

	ctx := &channelContext{br.done}

	// Initial fetch cycle.
	br.fetchAndPublish(ctx)
	br.module.UpdateStatus(status.Running, fmt.Sprintf("batched runner for %s is running", br.module.Name()))

	t := time.NewTicker(br.module.Config().Period)
	defer t.Stop()

	for {
		select {
		case <-br.done:
			return
		case <-t.C:
			br.fetchAndPublish(ctx)
		}
	}
}

// fetchAndPublish fetches all metricsets in parallel, buffering events in
// per-metricset reporters, then publishes each metricset's events as a batch.
func (br *batchedRunner) fetchAndPublish(ctx *channelContext) {
	reporters := make([]*bufferingReporter, len(br.msws))
	for i := range br.msws {
		reporters[i] = &bufferingReporter{
			msw:  br.msws[i],
			done: br.done,
		}
	}

	// Fetch all metricsets in parallel.
	var wg sync.WaitGroup
	wg.Add(len(br.msws))
	for i, msw := range br.msws {
		go func(i int, msw *metricSetWrapper) {
			defer wg.Done()
			defer msw.Logger().Recover(fmt.Sprintf("recovered from panic while fetching "+
				"'%s/%s' for host '%s'", msw.module.Name(), msw.Name(), msw.Host()))
			reporters[i].StartFetchTimer()
			msw.fetch(ctx, reporters[i])
		}(i, msw)
	}
	wg.Wait()

	// Publish each metricset's buffered events as a batch.
	for i, r := range reporters {
		if events := r.flush(); len(events) > 0 {
			br.clients[i].PublishAll(events)
		}
	}
}

func (br *batchedRunner) String() string {
	return fmt.Sprintf("%s [batched, metricsets=%d]", br.module.Name(), len(br.msws))
}

func (br *batchedRunner) SetStatusReporter(reporter status.StatusReporter) {
	// All metricSetWrappers in the batched runner belong to the same module,
	// so setting the status reporter on any one of them applies to all.
	if msw := br.msws; len(msw) > 0 {
		msw[0].module.SetStatusReporter(reporter)
	}
}

func (br *batchedRunner) Diagnostics() []diagnostics.DiagnosticSetup {
	var responses []diagnostics.DiagnosticSetup
	for _, msw := range br.msws {
		diagHandler, ok := msw.MetricSet.(diagnostics.DiagnosticReporter)
		if !ok {
			continue
		}
		diags := diagHandler.Diagnostics()
		pathPrefix := fmt.Sprintf("%s-%s", br.module.Name(), msw.MetricSet.Name())
		for _, diag := range diags {
			diag.Filename = filepath.Join(pathPrefix, diag.Filename)
			responses = append(responses, diag)
		}
	}
	return responses
}
