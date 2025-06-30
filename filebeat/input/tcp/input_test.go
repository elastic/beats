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
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
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

	publisher := newMockPublisher(t)
	startTCPClient(t, 2*time.Second, serverAddr, []string{"foo", "bar"})

	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		ID:          t.Name(),
		Cancelation: ctx,
		Logger:      logp.NewNopLogger(),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := inp.Run(v2Ctx, publisher); err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Errorf("input exited with error: %s", err)
			}
		}
	}()

	require.Eventually(
		t,
		func() bool {
			return publisher.count.Load() == 2
		},
		5*time.Second,
		100*time.Millisecond,
		"not all events published")

	// Assert metrics
	m := getEventMetrics(t)
	if got, want := m.EventsPublished, 2; got != want {
		t.Errorf("expecting %d events published, got %d", want, got)
	}

	if got, want := m.EventsRead, 2; got != want {
		t.Errorf("expecting %d events read, got %d", want, got)
	}

	// Stop the input, this removes all metrics
	cancel()

	// Ensure the input Run method returns
	wg.Wait()
}

func TestInputCanReadWithoutPublishing(t *testing.T) {
	serverAddr := "localhost:9042"
	wg := sync.WaitGroup{}
	inp, err := configure(conf.MustNewConfigFrom(map[string]any{
		"host": serverAddr,
	}))
	if err != nil {
		t.Fatalf("cannot create input: %s", err)
	}

	publisher := newMockPublisher(t)
	publisher.blocked.Store(true)
	startTCPClient(t, 2*time.Second, serverAddr, []string{"foo", "bar"})

	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		ID:          t.Name(),
		Cancelation: ctx,
		Logger:      logp.NewNopLogger(),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := inp.Run(v2Ctx, publisher); err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Errorf("input exited with error: %s", err)
			}
		}
	}()

	require.Eventually(
		t,
		func() bool {
			m := getEventMetrics(t)
			return m.EventsRead == 2 && m.EventsPublished == 0
		},
		time.Second,
		100*time.Millisecond,
		"did not find 2 events read and 0 published")

	// Stop the input
	cancel()

	// Ensure the input Run method returns
	wg.Wait()
}

func newMockPublisher(t *testing.T) *mockPublisher {
	return &mockPublisher{
		t:       t,
		count:   atomic.Uint64{},
		blocked: atomic.Bool{},
	}
}

type mockPublisher struct {
	t       *testing.T
	count   atomic.Uint64
	blocked atomic.Bool
}

func (m *mockPublisher) Publish(evt beat.Event) {
	for m.blocked.Load() {
		time.Sleep(10 * time.Millisecond)
	}

	m.count.Add(1)
}

func startTCPClient(t *testing.T, timeout time.Duration, address string, dataToSend []string) {
	go func() {
		var conn net.Conn
		var err error

		// Keep trying to connect to the server with a timeout
		ticker := time.Tick(100 * time.Millisecond)
		timer := time.After(timeout)
	FOR:
		for {
			select {
			case <-ticker:
				conn, err = net.Dial("tcp", address)
				if err == nil {
					break FOR
				}
			case <-timer:
				t.Errorf("could not connect to %s after %s", address, timeout)
				return
			}
		}

		defer conn.Close()

		// Send data to the server
		for _, data := range dataToSend {
			_, err := conn.Write([]byte(data + "\n"))
			if err != nil {
				t.Errorf("Failed to send data: %s", err)
				return
			}
			time.Sleep(100 * time.Millisecond) // Simulate delay between messages
		}
	}()
}

type eventMetrics struct {
	EventsPublished int `json:"events_published"`
	EventsRead      int `json:"events_read"`
}

func getEventMetrics(t *testing.T) eventMetrics {
	data, err := inputmon.MetricSnapshotJSON(nil)
	if err != nil {
		t.Fatalf("cannot get metrics snapshot: %s", err)
	}

	metrics := []eventMetrics{}

	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("cannot read metrics: %s", err)
	}

	if len(metrics) == 0 {
		return eventMetrics{}
	}

	return metrics[0]
}
