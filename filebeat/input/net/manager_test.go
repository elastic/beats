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

package net

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/testpipeline"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestPublishLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: ctx,
	}

	w := wrapper{
		evtChan: make(chan beat.Event),
	}

	publisher := testpipeline.NewPipelineConnector()
	client, _ := publisher.Connect()
	metrics := &metricsMock{
		EventPublishedFunc: func(start time.Time) {},
		EventReceivedFunc:  func(len int, timestamp time.Time) {},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.publishLoop(v2Ctx, 0, client, metrics)
	}()

	events := []beat.Event{
		{Fields: mapstr.M{"message": "test1"}},
		{Fields: mapstr.M{"message": "test2"}},
	}

	for _, evt := range events {
		w.evtChan <- evt
	}

	assert.Eventuallyf(t, func() bool {
		return int(publisher.EventsPublished()) == len(events)
	},
		time.Second,
		100*time.Millisecond,
		"not all %d events have been published",
		len(events),
	)

	// Stop the publish loop and wait for it to exit
	cancel()
	wg.Wait()

	cc, ok := client.(*testpipeline.Client)
	if !ok {
		t.Fatalf("pipeline client is not '*testpipeline.Client', got %T", client)
	}

	if !cc.Closed() {
		t.Fatal("publishLoop did not call Close on the client")
	}
}

func TestInitWorkers(t *testing.T) {
	expectedClients := 2
	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: ctx,
	}

	w := wrapper{
		evtChan:            make(chan beat.Event),
		NumPipelineWorkers: expectedClients,
	}

	publisher := testpipeline.NewPipelineConnector()
	metrics := &metricsMock{
		EventPublishedFunc: func(start time.Time) {},
		EventReceivedFunc:  func(len int, timestamp time.Time) {},
	}

	if err := w.initWorkers(v2Ctx, publisher, metrics); err != nil {
		t.Fatalf("did not expect an error from initWorkers: %s", err)
	}

	if want, got := expectedClients, publisher.NumClients(); want != got {
		t.Fatalf("not all clients have been started. Want %d got %d", want, got)
	}

	cancel()
}

func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: ctx,
	}

	wg := sync.WaitGroup{}
	runCalled := atomic.Bool{}
	wg.Add(2)
	metrics := &metricsMock{
		EventPublishedFunc: func(start time.Time) {},
		EventReceivedFunc:  func(len int, timestamp time.Time) {},
	}
	w := wrapper{
		evtChan:            make(chan beat.Event),
		NumPipelineWorkers: 2,
		inp: &inputMock{
			NameFunc:        func() string { return t.Name() },
			InitMetricsFunc: func(s string, logger *logp.Logger) Metrics { return metrics },
			RunFunc: func(context v2.Context, eventCh chan<- beat.Event, metrics Metrics) error {
				runCalled.Store(true)
				defer wg.Done()
				select {
				case <-context.Cancelation.Done():
					return nil
				}
			},
		},
	}

	publisher := testpipeline.NewPipelineConnector()

	go func() {
		defer wg.Done()
		if err := w.Run(v2Ctx, publisher); err != nil {
			t.Errorf("Run returned with error: %s", err)
		}
	}()

	require.Eventually(
		t,
		func() bool {
			return runCalled.Load()
		},
		time.Second,
		100*time.Millisecond,
		"Run method from input has not been called",
	)

	cancel()
	wg.Wait()
}

func TestRunRecoversFromPanic(t *testing.T) {
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: t.Context(),
	}

	metrics := &metricsMock{
		EventPublishedFunc: func(start time.Time) {},
		EventReceivedFunc:  func(len int, timestamp time.Time) {},
	}
	inputName := "TCP&UDP"
	w := wrapper{
		evtChan:            make(chan beat.Event),
		NumPipelineWorkers: 1,
		inp: &inputMock{
			NameFunc:        func() string { return inputName },
			InitMetricsFunc: func(s string, logger *logp.Logger) Metrics { return metrics },
			RunFunc: func(context v2.Context, eventCh chan<- beat.Event, metrics Metrics) error {
				panic("can I recover?")
			},
		},
	}

	publisher := testpipeline.NewPipelineConnector()

	err := w.Run(v2Ctx, publisher)
	if err == nil {
		t.Fatal("expecting an error")
	}

	errMsg := err.Error()
	prefix := fmt.Sprintf("%s input panic", inputName)
	if !strings.HasPrefix(errMsg, prefix) {
		t.Fatalf("expecting error message to start with %q, but got: %q", prefix, errMsg)
	}
}

func TestRunReturnsInitWokersError(t *testing.T) {
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: t.Context(),
	}

	metrics := &metricsMock{
		EventPublishedFunc: func(start time.Time) {},
		EventReceivedFunc:  func(len int, timestamp time.Time) {},
	}

	w := wrapper{
		evtChan:            make(chan beat.Event),
		NumPipelineWorkers: 1,
		inp: &inputMock{
			NameFunc:        func() string { return t.Name() },
			InitMetricsFunc: func(s string, logger *logp.Logger) Metrics { return metrics },
		},
	}

	expectedErr := errors.New("oops")
	publisher := testpipeline.NewPipelineConnectorWithError(expectedErr)

	err := w.Run(v2Ctx, publisher)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expecting error containing %q to be returned, got %q", expectedErr, err)
	}
}
