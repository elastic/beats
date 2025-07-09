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

type NetInputMetrics struct {
	Packets   int `json:"received_events_total"`
	Published int `json:"published_events_total"`
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

func RequireNetInputMetrics(t *testing.T, timeout time.Duration, want NetInputMetrics) {
	msg := &strings.Builder{}
	require.Eventuallyf(
		t,
		func() bool {
			msg.Reset()
			got := GetNetInputMetrics(t)
			fmt.Fprintf(
				msg,
				"%d packets (events), %d events published",
				got.Packets,
				got.Published,
			)
			return got.Published == want.Published && got.Packets == want.Packets
		},
		timeout,
		100*time.Millisecond,
		"expecting %d packets (events) read, %d published. Got %s",
		want.Packets,
		want.Published,
		msg)
}
