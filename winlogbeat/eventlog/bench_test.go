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

//go:build windows
// +build windows

package eventlog

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"golang.org/x/sys/windows/svc/eventlog"

	"github.com/menderesk/beats/v7/libbeat/common"
)

const gigabyte = 1 << 30

var (
	benchTest    = flag.Bool("benchtest", false, "Run benchmarks for the eventlog package.")
	injectAmount = flag.Int("inject", 1e6, "Number of events to inject before running benchmarks.")
)

// TestBenchmarkRead benchmarks each event log reader implementation with
// different batch sizes.
//
// Recommended usage:
//   go test -run TestBenchmarkRead -benchmem -benchtime 10s -benchtest -v .
func TestBenchmarkRead(t *testing.T) {
	if !*benchTest {
		t.Skip("-benchtest not enabled")
	}

	writer, teardown := createLog(t)
	defer teardown()

	setLogSize(t, providerName, gigabyte)

	// Publish test messages:
	for i := 0; i < *injectAmount; i++ {
		safeWriteEvent(t, writer, eventlog.Info, uint32(rand.Int63()%1000), []string{strconv.Itoa(i) + " " + randomSentence(256)})
	}

	for _, api := range []string{winEventLogAPIName, winEventLogExpAPIName} {
		t.Run("api="+api, func(t *testing.T) {
			for _, batchSize := range []int{10, 100, 500, 1000} {
				t.Run(fmt.Sprintf("batch_size=%d", batchSize), func(t *testing.T) {
					result := testing.Benchmark(benchmarkEventLog(api, batchSize))
					outputBenchmarkResults(t, result)
				})
			}
		})
	}
}

func benchmarkEventLog(api string, batchSize int) func(b *testing.B) {
	return func(b *testing.B) {
		conf := common.MapStr{
			"name":            providerName,
			"batch_read_size": batchSize,
			"no_more_events":  "stop",
		}

		log := openLog(b, api, nil, conf)
		defer log.Close()

		events := 0
		b.ResetTimer()

		// Each iteration reads one batch.
		for i := 0; i < b.N; i++ {
			records, err := log.Read()
			if err != nil {
				b.Fatal(err)
				return
			}
			events += len(records)
		}

		b.StopTimer()

		b.ReportMetric(float64(events), "events")
		b.ReportMetric(float64(batchSize), "batch_size")
	}
}

func outputBenchmarkResults(t testing.TB, result testing.BenchmarkResult) {
	totalBatches := result.N
	totalEvents := int(result.Extra["events"])
	totalBytes := result.MemBytes
	totalAllocs := result.MemAllocs

	eventsPerSec := float64(totalEvents) / result.T.Seconds()
	bytesPerEvent := float64(totalBytes) / float64(totalEvents)
	bytesPerBatch := float64(totalBytes) / float64(totalBatches)
	allocsPerEvent := float64(totalAllocs) / float64(totalEvents)
	allocsPerBatch := float64(totalAllocs) / float64(totalBatches)

	t.Logf("%.2f events/sec\t %d B/event\t %d B/batch\t %d allocs/event\t %d allocs/batch",
		eventsPerSec, int(bytesPerEvent), int(bytesPerBatch), int(allocsPerEvent), int(allocsPerBatch))
}

var randomWords = []string{
	"recover",
	"article",
	"highway",
	"bargain",
	"trolley",
	"college",
	"attract",
	"wriggle",
	"feather",
	"neutral",
	"percent",
	"quality",
	"manager",
	"hunting",
	"arrange",
}

func randomSentence(n uint) string {
	buf := bytes.NewBuffer(make([]byte, n))
	buf.Reset()

	for {
		idx := rand.Uint32() % uint32(len(randomWords))
		word := randomWords[idx]

		if buf.Len()+len(word) <= buf.Cap() {
			buf.WriteString(randomWords[idx])
		} else {
			break
		}

		if buf.Len()+1 <= buf.Cap() {
			buf.WriteByte(' ')
		} else {
			break
		}
	}

	return buf.String()
}
