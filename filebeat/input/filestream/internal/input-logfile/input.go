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
	"time"

	"github.com/elastic/beats/v7/filebeat/input/filestream/internal/task"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/go-concert/ctxtool"
)

type managedInput struct {
	userID           string
	metricsID        string
	manager          *InputManager
	ackCH            *updateChan
	sourceIdentifier *sourceIdentifier
	prospector       Prospector
	harvester        Harvester
	cleanTimeout     time.Duration
	harvesterLimit   uint64
}

// Name is required to implement the v2.Input interface
func (inp *managedInput) Name() string { return inp.harvester.Name() }

// Test runs the Test method for each configured source.
func (inp *managedInput) Test(ctx input.TestContext) error {
	return inp.prospector.Test()
}

// Run
func (inp *managedInput) Run(
	ctx input.Context,
	pipeline beat.PipelineConnector,
) (err error) {
	ctx.UpdateStatus(status.Starting, "")
	groupStore := inp.manager.getRetainedStore()
	defer groupStore.Release()

	// Setup cancellation using a custom cancel context. All workers will be
	// stopped if one failed badly by returning an error.
	cancelCtx, cancel := context.WithCancel(ctxtool.FromCanceller(ctx.Cancelation))
	defer cancel()
	ctx.Cancelation = cancelCtx

	metrics := NewMetrics(inp.metricsID)
	defer metrics.Close()

	hg := &defaultHarvesterGroup{
		pipeline:     pipeline,
		readers:      newReaderGroup(),
		cleanTimeout: inp.cleanTimeout,
		harvester:    inp.harvester,
		store:        groupStore,
		ackCH:        inp.ackCH,
		identifier:   inp.sourceIdentifier,
		tg: task.NewGroup(
			inp.harvesterLimit,
			time.Minute, // magic number
			ctx.Logger,
			"harvester:"),
		metrics: metrics,
	}

	prospectorStore := inp.manager.getRetainedStore()
	defer prospectorStore.Release()
	sourceStore := newSourceStore(prospectorStore, inp.sourceIdentifier)

	// Mark it as running for now.
	// Any errors encountered by harverter will change state to Degraded
	ctx.UpdateStatus(status.Running, "")

	inp.prospector.Run(ctx, sourceStore, hg)

	// Notify the manager the input has stopped, currently that is used to
	// keep track of duplicated IDs
	inp.manager.StopInput(inp.userID)

	return nil
}

func newInputACKHandler(ch *updateChan) beat.EventListener {
	return acker.EventPrivateReporter(func(acked int, private []interface{}) {
		var n uint
		var last int
		for i := 0; i < len(private); i++ {
			current := private[i]
			if current == nil {
				continue
			}

			if _, ok := current.(*updateOp); !ok {
				continue
			}

			n++
			last = i
		}

		if n == 0 {
			return
		}

		op := private[last].(*updateOp)
		ch.Send(scheduledUpdate{op: op, n: n})
	})
}
