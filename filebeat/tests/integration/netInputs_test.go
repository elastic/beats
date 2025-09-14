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

//go:build integration

package integration

import (
	_ "embed"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/filebeat/input/net/nettest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestNetInputs(t *testing.T) {
	testCases := map[string]struct {
		cfgFile     string
		data        []string
		runClientFn func(t *testing.T, addr string, data []string)
	}{
		"TCP": {
			cfgFile:     "tcp.yml",
			data:        []string{"foo", "bar"},
			runClientFn: nettest.RunTCPClient,
		},
		"UDP": {
			cfgFile:     "udp.yml",
			data:        []string{"foo", "bar"},
			runClientFn: nettest.RunUDPClient,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)

			// DO NOT USE 'localhost', the UDP client does work when using it.
			addr := "127.0.0.1:4242"
			cfg := getConfig(t, map[string]any{"addr": addr}, "netInputs", tc.cfgFile)

			filebeat.WriteConfigFile(cfg)
			filebeat.Start()
			filebeat.WaitLogsContainsAnyOrder(
				[]string{
					"[Worker 0] starting publish loop",
					"[Worker 1] starting publish loop",
				},
				20*time.Second,
				"not all workers have started",
			)

			tc.runClientFn(t, addr, tc.data)

			filebeat.WaitPublishedEvents(3*time.Second, len(tc.data))
			filebeat.Stop()
			filebeat.WaitLogsContainsAnyOrder(
				[]string{
					"[Worker 0] finished publish loop",
					"[Worker 1] finished publish loop",
				},
				5*time.Second,
				"not all workers have started",
			)
		})
	}
}

func TestNetInputsCanReadWithBlockedOutput(t *testing.T) {
	testCases := map[string]struct {
		cfgFile     string
		input       string
		events      int
		expectedInQ int
		numWorkers  int
		runClientFn func(t *testing.T, addr string, data []string)
	}{
		"TCP": {
			cfgFile:     "es.yml",
			input:       "tcp",
			events:      500, // That needs to be more than can be published
			numWorkers:  5,
			runClientFn: nettest.RunTCPClient,
		},
		"UDP": {
			cfgFile:     "es.yml",
			input:       "udp",
			events:      500, // That needs to be more than can be published
			numWorkers:  5,
			runClientFn: nettest.RunUDPClient,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)

			id := uuid.Must(uuid.NewV4())
			data := []string{}
			for range tc.events {
				data = append(data, strings.Repeat("FooBar", 50))
			}
			workerStartedMsgs := []string{}
			workerDoneMsgs := []string{}
			for i := range tc.numWorkers {
				workerStartedMsgs = append(workerStartedMsgs, fmt.Sprintf("[Worker %d] starting publish loop", i))
				workerDoneMsgs = append(workerDoneMsgs, fmt.Sprintf("[Worker %d] finished publish loop", i))
			}

			esServer, esAddr, _, _ := integration.StartMockES(t, "", 0, 0, 0, 0, 0)
			defer esServer.Close()
			proxy, proxyURL := integration.NewDisablingProxy(t, esAddr)
			proxy.Disable()

			// DO NOT USE 'localhost', the UDP client does work when using it.
			addr := "127.0.0.1:4242"
			cfg := getConfig(t, map[string]any{
				"id":         id,
				"input":      tc.input,
				"addr":       addr,
				"esHost":     proxyURL,
				"numWorkers": tc.numWorkers,
			}, "netInputs", tc.cfgFile)

			filebeat.WriteConfigFile(cfg)
			filebeat.Start()
			filebeat.WaitLogsContainsAnyOrder(
				workerStartedMsgs,
				5*time.Second,
				"not all workers have started",
			)

			tc.runClientFn(t, addr, data)

			// Ensure the events are in the publishing pipeline.
			// The events are logged when they enter the publishing pipeline.
			// the events in the publishing pipeline are the queue size + the
			// number of pipeline workers
			expectedEvents := 32 + tc.numWorkers
			filebeat.WaitEventsInLogFile(expectedEvents, 3*time.Second)

			// Ensure the output is not accepting events
			filebeat.WaitLogsContains(
				"Ping request failed with: 503 Service Unavailable: Proxy is disabled",
				time.Second,
				"cannot find output error in the logs")

			m := nettest.GetHTTPInputMetrics(t, id.String(), "http://127.0.0.1:5066")

			// The number of events published is equal to the queue size
			expectPublished := 32
			// The number of events read by the input is:
			// queue size + (1 event per pipeline worker)*(NumCPU +1) + 2 for
			// the input goroutine.
			//
			// The number of pipeline workers is multiplied by 6 because the channel
			// used for the input and worker goroutines is sized as:
			// num_workers * runtime.NumCPU() + 1, then we need to add an extra
			// event per worker goroutine running.
			expectedEventsRead := 32 + tc.numWorkers*(runtime.NumCPU()+1) + 2

			if m.PublishedEventsTotal != expectPublished {
				t.Errorf(
					"expecting input metric 'published_events_total' to be %d, but got %d",
					expectPublished,
					m.PublishedEventsTotal)
			}

			if m.ReceivedEventsTotal != expectedEventsRead {
				t.Errorf(
					"expecting input metric 'received_events_total' to be %d, but got %d",
					expectedEventsRead,
					m.ReceivedEventsTotal)
			}

			filebeat.Stop()

			// Ensure all workers are finished, no goroutine leak
			filebeat.WaitLogsContainsAnyOrder(
				workerDoneMsgs,
				5*time.Second,
				"not all workers have started",
			)
		})
	}
}
