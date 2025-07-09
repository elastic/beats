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

package inputtest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
)

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

// RequireNetMetricsCount uses require.Eventually to ensure the received and
// published counts are the same.
// It assets the following values:
//   - received_bytes_total
//   - published_events_total
//   - processing_time count
//   - arrival_period count
func RequireNetMetricsCount(t *testing.T, timeout time.Duration, received, published int) {
	want := NetInputMetrics{
		ReceivedEventsTotal:  received,
		PublishedEventsTotal: published,
		ArrivalPeriod:        ArrivalPeriod{Histogram: Histogram{Count: received - 1}},
		ProcessingTime:       ProcessingTime{Histogram: Histogram{Count: published}},
	}

	msg := &strings.Builder{}
	require.Eventuallyf(
		t,
		func() bool {
			msg.Reset()
			got := GetNetInputMetrics(t)
			fmt.Fprintf(
				msg,
				"received: %d, published: %d, arrival_period count: %d, processing_time count: %d",
				got.ReceivedEventsTotal,
				got.PublishedEventsTotal,
				got.ArrivalPeriod.Histogram.Count,
				got.ProcessingTime.Histogram.Count,
			)

			return got.PublishedEventsTotal == want.PublishedEventsTotal &&
				got.ReceivedEventsTotal == want.ReceivedEventsTotal &&
				got.ArrivalPeriod.Histogram.Count == want.ArrivalPeriod.Histogram.Count &&
				got.ProcessingTime.Histogram.Count == want.ProcessingTime.Histogram.Count
		},
		timeout,
		100*time.Millisecond,
		"expecting received: %d, published: %d, arrival_period count: %d, processing_time count: %d. Got %s",
		want.ReceivedEventsTotal,
		want.PublishedEventsTotal,
		want.ArrivalPeriod.Histogram.Count,
		want.ProcessingTime.Histogram.Count,
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
