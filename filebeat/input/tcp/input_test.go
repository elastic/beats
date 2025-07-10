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

//This file was contributed to by generative AI

package tcp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestInput(t *testing.T) {
	serverAddr := "localhost:9042"
	wg := sync.WaitGroup{}
	inp, err := configure(conf.MustNewConfigFrom(map[string]any{
		"host": serverAddr,
	}))
	if err != nil {
		t.Fatalf("cannot create input: %s", err)
	}

	data := []string{"foo", "bar"}
	go inputtest.RunTCPClient(t, serverAddr, data)

	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		ID:          t.Name(),
		Cancelation: ctx,
		Logger:      logp.NewNopLogger(),
	}

	metrics := inp.InitMetrics("tcp", v2Ctx.Logger)
	c := make(chan beat.Event, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := inp.Run(v2Ctx, c, metrics); err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Errorf("input exited with error: %s", err)
			}
		}
	}()

	inputtest.RequireNetMetricsCount(t, time.Second, 2, 0, 6)

	// Stop the input, this removes all metrics
	cancel()

	// Ensure the input Run method returns
	wg.Wait()

	// Make sure all events have been written to the channel
	evtCount := 0
	for range len(data) {
		select {
		case <-c:
			evtCount++
		default:
			t.Fatalf("only %d events have been written to the channel, expecting %d", evtCount, len(data))
		}
	}

	select {
	case <-c:
		t.Fatalf("expecting %d events on the channel, got at least %d", len(data), evtCount+1)
	default:
		// No more events on the channel, test passed
	}
}

// func TestInputCanReadWithoutPublishing(t *testing.T) {
// 	serverAddr := "localhost:9042"
// 	numberOfWorkers := 42
// 	wg := sync.WaitGroup{}
// 	inp, err := configure(conf.MustNewConfigFrom(map[string]any{
// 		"host":              serverAddr,
// 		"number_of_workers": numberOfWorkers,
// 	}))
// 	if err != nil {
// 		t.Fatalf("cannot create input: %s", err)
// 	}

// 	pipeline := testpipeline.NewPipelineConnector()
// 	pipeline.Block()

// 	go inputtest.RunTCPClient(t, serverAddr, []string{"foo", "bar"})

// 	ctx, cancel := context.WithCancel(t.Context())
// 	v2Ctx := v2.Context{
// 		ID:          t.Name(),
// 		Cancelation: ctx,
// 		Logger:      logp.NewNopLogger(),
// 	}

// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		if err := inp.Run(v2Ctx, pipeline); err != nil {
// 			if !errors.Is(err, context.Canceled) {
// 				t.Errorf("input exited with error: %s", err)
// 			}
// 		}
// 	}()

// 	inputtest.RequireNetMetricsCount(t, time.Second, 2, 0, 6)
// 	// Stop the input
// 	cancel()

// 	// Ensure the input Run method returns
// 	wg.Wait()

// 	if got, want := pipeline.NumClients(), numberOfWorkers; got != want {
// 		t.Fatalf("did not create the expected number of clients, expecting %d, got %d", want, got)
// 	}
// }
