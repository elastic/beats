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

package tcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	netinput "github.com/elastic/beats/v7/filebeat/input/net"
	"github.com/elastic/beats/v7/filebeat/input/net/nettest"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInput(t *testing.T) {
	serverAddr := "localhost:9042"
	wg := sync.WaitGroup{}
	inp, err := configure(conf.MustNewConfigFrom(map[string]any{
		"host":              serverAddr,
		"number_of_workers": 2,
	}))
	if err != nil {
		t.Fatalf("cannot create input: %s", err)
	}

	data := []string{"foo", "bar"}
	go nettest.RunTCPClient(t, serverAddr, data)

	ctx, cancel := context.WithCancel(t.Context())
	v2Ctx := v2.Context{
		ID:              t.Name(),
		Cancelation:     ctx,
		Logger:          logp.NewNopLogger(),
		MetricsRegistry: monitoring.NewRegistry(),
	}

	metrics := inp.InitMetrics("tcp", v2Ctx.MetricsRegistry, v2Ctx.Logger)
	c := make(chan netinput.DataMetadata, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := inp.Run(v2Ctx, c, metrics); err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Errorf("input exited with error: %s", err)
			}
		}
	}()

	nettest.RequireNetMetricsCount(t, v2Ctx.MetricsRegistry, time.Second, 2, 0, 6)

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

func BenchmarkInput(b *testing.B) {
	port, err := libbeattesting.AvailableTCP4Port()
	if err != nil {
		b.Fatalf("cannot find available port: %s", err)
	}
	serverAddr := net.JoinHostPort("localhost", fmt.Sprintf("%d", port))

	inp, err := configure(conf.MustNewConfigFrom(map[string]any{
		"host":              serverAddr,
		"number_of_workers": 2,
	}))
	if err != nil {
		b.Fatalf("cannot create input: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v2Ctx := v2.Context{
		ID:              b.Name(),
		Cancelation:     ctx,
		Logger:          logp.NewNopLogger(),
		MetricsRegistry: monitoring.NewRegistry(),
	}

	metrics := inp.InitMetrics("tcp", v2Ctx.MetricsRegistry, v2Ctx.Logger)
	c := make(chan netinput.DataMetadata, 5*runtime.NumCPU())

	go func() {
		if err := inp.Run(v2Ctx, c, metrics); err != nil {
			if !errors.Is(err, context.Canceled) {
				b.Errorf("input exited with error: %s", err)
			}
		}
	}()

	require.EventuallyWithTf(b, func(ct *assert.CollectT) {
		conn, err := net.Dial("tcp", serverAddr)
		require.NoError(ct, err)
		conn.Close()
	}, 30*time.Second, 100*time.Millisecond, "waiting for TCP server to start")

	testMessage := bytes.Repeat([]byte("A"), 1001)
	testMessage[len(testMessage)-1] = '\n'
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				b.Errorf("cannot create connection: %s", err)
				continue
			}

			for range 100 {
				_, err = conn.Write(testMessage)
				if err != nil {
					b.Errorf("failed to send data: %s", err)
					break
				}
			}
			conn.Close()

			// Read the events from the channel to prevent blocking
			for range 100 {
				select {
				case <-c:
				case <-time.After(time.Second):
					b.Error("timeout waiting for event")
				}
			}
		}
	})
}
