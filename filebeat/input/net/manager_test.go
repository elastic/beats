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

// This file was contributed to by generative AI

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
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestCreate(t *testing.T) {
	testCases := map[string]struct {
		config    *conf.C
		expectErr string
	}{
		"happy path": {
			config: conf.MustNewConfigFrom(map[string]any{
				"number_of_workers": 42,
			}),
		},
		"unpack error": {
			config: conf.MustNewConfigFrom(map[string]any{
				"number_of_workers": -1,
			}),
			expectErr: "negative value accessing 'number_of_workers'",
		},
		"configure error": {
			config: conf.MustNewConfigFrom(map[string]any{
				"number_of_workers": 42,
			}),
			expectErr: "oops",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			m := NewManager(func(c *conf.C) (Input, error) {
				if tc.expectErr != "" {
					return nil, errors.New(tc.expectErr)
				}

				return &inputMock{}, nil
			})

			inp, err := m.Create(tc.config)
			if tc.expectErr != "" && !strings.Contains(err.Error(), tc.expectErr) {
				t.Errorf("expecting Create to return error containing %q, got %q instead", tc.expectErr, err)
				if inp != nil {
					t.Error("on error the returned input must be nil")
				}
			}
		})
	}
}

func TestPublishLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: ctx,
	}

	w := wrapper{
		evtChan: make(chan DataMetadata),
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

	events := [][]byte{
		[]byte("test1"),
		[]byte("test2"),
	}

	for _, msg := range events {
		w.evtChan <- DataMetadata{
			Timestamp: time.Now(),
			Data:      msg,
		}
	}

	assert.Eventuallyf(t, func() bool {
		//nolint:gosec // it is a test, there is no risk of overflow
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
	v2Ctx := v2.Context{
		Logger:      logp.NewNopLogger(),
		Cancelation: t.Context(),
	}

	w := wrapper{
		evtChan:            make(chan DataMetadata),
		numPipelineWorkers: expectedClients,
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
		evtChan:            make(chan DataMetadata),
		numPipelineWorkers: 2,
		inp: &inputMock{
			NameFunc:        func() string { return t.Name() },
			InitMetricsFunc: func(s string, reg *monitoring.Registry, logger *logp.Logger) Metrics { return metrics },
			RunFunc: func(context v2.Context, eventCh chan<- DataMetadata, metrics Metrics) error {
				runCalled.Store(true)
				defer wg.Done()
				// Block until the context is cancelled
				<-context.Cancelation.Done()
				return nil
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
		evtChan:            make(chan DataMetadata),
		numPipelineWorkers: 1,
		inp: &inputMock{
			NameFunc:        func() string { return inputName },
			InitMetricsFunc: func(s string, reg *monitoring.Registry, logger *logp.Logger) Metrics { return metrics },
			RunFunc: func(context v2.Context, eventCh chan<- DataMetadata, metrics Metrics) error {
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
		evtChan:            make(chan DataMetadata),
		numPipelineWorkers: 1,
		inp: &inputMock{
			NameFunc:        func() string { return t.Name() },
			InitMetricsFunc: func(s string, reg *monitoring.Registry, logger *logp.Logger) Metrics { return metrics },
		},
	}

	expectedErr := errors.New("oops")
	publisher := testpipeline.NewPipelineConnectorWithError(expectedErr)

	err := w.Run(v2Ctx, publisher)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expecting error containing %q to be returned, got %q", expectedErr, err)
	}
}

func TestNameReturnsInputName(t *testing.T) {
	inpName := "foo bar"
	inp := &inputMock{
		NameFunc: func() string { return inpName },
	}
	w := wrapper{
		inp: inp,
	}

	got := w.Name()
	if got != inpName {
		t.Fatalf("expecting wrapper.Name to return w.inp.Name. Got %q instead of %q ", got, inpName)
	}
}

func TestWrapperTest(t *testing.T) {
	testCases := []struct {
		name      string
		testError error
	}{
		{
			name:      "successful test",
			testError: nil,
		},
		{
			name:      "test with error",
			testError: errors.New("oops"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockInput := &inputMock{
				TestFunc: func(testContext v2.TestContext) error {
					return tc.testError
				},
			}

			w := wrapper{
				inp: mockInput,
			}

			v2Ctx := v2.TestContext{
				Agent: beat.Info{
					Beat: "something to compare"},
			}

			err := w.Test(v2Ctx)
			if tc.testError != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tc.testError)
				} else if err.Error() != tc.testError.Error() {
					t.Errorf("Expected error %v, got %v", tc.testError, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if calls := len(mockInput.TestCalls()); calls != 1 {
				t.Errorf("Expected TestFunc to be called exactly once, got %d calls", calls)
			}

			// Verify the test context was passed correctly
			if len(mockInput.TestCalls()) > 0 {
				if v2Ctx != mockInput.TestCalls()[0].TestContext {
					t.Fatal("wrong v2 context passed when calling Test")
				}
			}
		})
	}
}
