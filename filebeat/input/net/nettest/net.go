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

package nettest

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
)

// RunUDPClient sends each string in `data` to address using a UDP connection.
// A new line ('\n') is added at the end of each entry.
// There is a 100ms delay between sends, errors are logged, but not retried.
func RunUDPClient(t *testing.T, address string, data []string) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		t.Errorf("cannot create connection: %s", err)
	}
	defer conn.Close()

	// Send data to the server
	for _, data := range data {
		_, err = conn.Write([]byte(data + "\n"))
		if err != nil {
			t.Logf("Error sending data: %s, skipping to next entry", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// RunTCPClient sends each string in `data` to address using a TCP connection.
// It re-tries opening the connection for 5 seconds with 100ms delay.
// A new line ('\n') is added at the end of each entry.
// There is a 100ms delay between sends. Failing to send will fail the test
func RunTCPClient(t *testing.T, address string, data []string) {
	var conn net.Conn
	var err error

	// Keep trying to connect to the server with a timeout
	ticker := time.Tick(100 * time.Millisecond)
	timer := time.After(5 * time.Second)
FOR:
	for {
		select {
		case <-ticker:
			conn, err = net.Dial("tcp", address)
			if err == nil {
				break FOR
			}
		case <-timer:
			t.Errorf("could not connect to %s after 5s", address)
			return
		}
	}

	defer conn.Close()

	// Send data to the server
	for _, data := range data {
		_, err := conn.Write([]byte(data + "\n"))
		if err != nil {
			t.Errorf("Failed to send data: %s", err)
			return
		}
		time.Sleep(100 * time.Millisecond) // Simulate delay between messages
	}
}

func GetNetInputMetrics(t *testing.T) NetInputMetrics {
	data, err := inputmon.MetricSnapshotJSON(nil)
	if err != nil {
		t.Fatalf("cannot get metrics snapshot: %s", err)
	}

	metrics := []NetInputMetrics{}
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("cannot read metrics: %s", err)
	}

	if len(metrics) == 0 {
		return NetInputMetrics{}
	}

	return metrics[0]
}

// RequireNetMetricsCount uses require.Eventually to assert
// the following values:
//   - received_bytes_total
//   - published_events_total
//   - processing_time count
//   - arrival_period count
//   - received_bytes_total
func RequireNetMetricsCount(t *testing.T, timeout time.Duration, received, published, bytes int) {
	t.Helper()
	want := NetInputMetrics{
		ReceivedEventsTotal:  received,
		PublishedEventsTotal: published,
		ArrivalPeriod:        ArrivalPeriod{Histogram: Histogram{Count: received - 1}},
		ProcessingTime:       ProcessingTime{Histogram: Histogram{Count: published}},
		ReceivedBytesTotal:   bytes,
	}

	msg := &strings.Builder{}
	require.Eventuallyf(
		t,
		func() bool {
			msg.Reset()
			got := GetNetInputMetrics(t)
			fmt.Fprintf(
				msg,
				"received: %d, published: %d, arrival_period count: %d, "+
					"processing_time count: %d, bytes: %d",
				got.ReceivedEventsTotal,
				got.PublishedEventsTotal,
				got.ArrivalPeriod.Histogram.Count,
				got.ProcessingTime.Histogram.Count,
				got.ReceivedBytesTotal,
			)

			return got.PublishedEventsTotal == want.PublishedEventsTotal &&
				got.ReceivedEventsTotal == want.ReceivedEventsTotal &&
				got.ArrivalPeriod.Histogram.Count == want.ArrivalPeriod.Histogram.Count &&
				got.ProcessingTime.Histogram.Count == want.ProcessingTime.Histogram.Count &&
				got.ReceivedBytesTotal == want.ReceivedBytesTotal
		},
		timeout,
		100*time.Millisecond,
		"expecting received: %d, published: %d, arrival_period count: %d, "+
			"processing_time count: %d, bytes: %d. Got %s",
		want.ReceivedEventsTotal,
		want.PublishedEventsTotal,
		want.ArrivalPeriod.Histogram.Count,
		want.ProcessingTime.Histogram.Count,
		want.ReceivedBytesTotal,
		msg)
}

type NetInputMetrics struct {
	ArrivalPeriod            ArrivalPeriod  `json:"arrival_period"`
	Device                   string         `json:"device"`
	ID                       string         `json:"id"`
	Input                    string         `json:"input"`
	ProcessingTime           ProcessingTime `json:"processing_time"`
	PublishedEventsTotal     int            `json:"published_events_total"`
	ReceiveQueueLength       int            `json:"receive_queue_length"`
	ReceivedBytesTotal       int            `json:"received_bytes_total"`
	ReceivedEventsTotal      int            `json:"received_events_total"`
	SystemPacketDrops        int            `json:"system_packet_drops"`
	UDPReadBufferLengthGauge int            `json:"udp_read_buffer_length_gauge"`
}

type Histogram struct {
	Count  int     `json:"count"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	Min    float64 `json:"min"`
	P75    float64 `json:"p75"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	P999   float64 `json:"p999"`
	Stddev float64 `json:"stddev"`
}

type ArrivalPeriod struct {
	Histogram Histogram `json:"histogram"`
}

type ProcessingTime struct {
	Histogram Histogram `json:"histogram"`
}
