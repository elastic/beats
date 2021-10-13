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

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"
)

func TestGetCounter(t *testing.T) {
	reader := Reader{
		query: pdh.Query{},
		log:   nil,
		counters: []PerfCounter{
			{
				QueryField:   "datagrams_sent_per_sec",
				QueryName:    `\UDPv4\Datagrams Sent/sec`,
				Format:       "float",
				ObjectName:   "UDPv4",
				ObjectField:  "object",
				ChildQueries: []string{`\UDPv4\Datagrams Sent/sec`},
			},
		},
	}
	ok, val := reader.getCounter(`\UDPv4\Datagrams Sent/sec`)
	assert.True(t, ok)
	assert.Equal(t, val.QueryField, "datagrams_sent_per_sec")
	assert.Equal(t, val.ObjectName, "UDPv4")

}

func TestMapCounters(t *testing.T) {
	config := Config{
		IgnoreNECounters:  false,
		GroupMeasurements: false,
		Queries: []Query{
			{
				Name:      "Process",
				Namespace: "metrics",
				Instance:  []string{"svchost*"},
				Counters: []QueryCounter{
					{
						Name:   "% Processor Time",
						Format: "float",
					},
				},
			},
			{
				Name:      "Process",
				Field:     "disk",
				Namespace: "metrics",
				Instance:  []string{"conhost*"},
				Counters: []QueryCounter{
					{
						Name:   "IO Read Operations/sec",
						Field:  "read_ops",
						Format: "double",
					},
				},
			},
		},
	}
	reader := Reader{}
	reader.mapCounters(config)
	assert.Equal(t, len(reader.counters), 2)
	for _, readerCounter := range reader.counters {
		if readerCounter.InstanceName == "svchost*" {
			assert.Equal(t, readerCounter.ObjectName, "Process")
			assert.Equal(t, readerCounter.ObjectField, "object")
			assert.Equal(t, readerCounter.QueryField, "metrics.%_processor_time")
			assert.Equal(t, readerCounter.QueryName, `\Process(svchost*)\% Processor Time`)
			assert.Equal(t, len(readerCounter.ChildQueries), 0)
			assert.Equal(t, readerCounter.Format, "float")
		} else {
			assert.Equal(t, readerCounter.InstanceName, "conhost*")
			assert.Equal(t, readerCounter.ObjectName, "Process")
			assert.Equal(t, readerCounter.ObjectField, "disk")
			assert.Equal(t, readerCounter.QueryField, "metrics.read_ops")
			assert.Equal(t, readerCounter.QueryName, `\Process(conhost*)\IO Read Operations/sec`)
			assert.Equal(t, len(readerCounter.ChildQueries), 0)
			assert.Equal(t, readerCounter.Format, "double")
		}
	}
}

func TestMapQuery(t *testing.T) {
	//mapQuery(obj string, instance string, path string) string {
	obj := "Process"
	instance := "*"
	path := "% Processor Time"
	result := mapQuery(obj, instance, path)
	assert.Equal(t, result, `\Process(*)\% Processor Time`)

	obj = `\Process\`
	instance = "(*"
	result = mapQuery(obj, instance, path)
	assert.Equal(t, result, `\Process(*)\% Processor Time`)
}

func TestMapCounterPathLabel(t *testing.T) {
	result := mapCounterPathLabel("metrics", "", `WININET: Bytes from server`)
	assert.Equal(t, result, "metrics.wininet_bytes_from_server")
	result = mapCounterPathLabel("metrics", "", `RSC Coalesced Packet Bucket 5 (16To31)`)
	assert.Equal(t, result, "metrics.rsc_coalesced_packet_bucket_5_(16_to31)")
	result = mapCounterPathLabel("metrics", "", `Total Memory Usage --- Non-Paged Pool`)
	assert.Equal(t, result, "metrics.total_memory_usage_---_non-paged_pool")
	result = mapCounterPathLabel("metrics", "", `IPv6 NBLs/sec indicated with low-resource flag`)
	assert.Equal(t, result, "metrics.ipv6_nbls_per_sec_indicated_with_low-resource_flag")
	result = mapCounterPathLabel("metrics", "", `Queued Poison Messages Per Second`)
	assert.Equal(t, result, "metrics.queued_poison_messages_per_second")
	result = mapCounterPathLabel("metrics", "", `I/O Log Writes Average Latency`)
	assert.Equal(t, result, "metrics.i/o_log_writes_average_latency")
	result = mapCounterPathLabel("metrics", "io.logwrites.average latency", `I/O Log Writes Average Latency`)
	assert.Equal(t, result, "metrics.io_logwrites_average_latency")

	result = mapCounterPathLabel("metrics", "this.is__exceptional-test:case/sec", `RSC Coalesced Packet Bucket 5 (16To31)`)
	assert.Equal(t, result, "metrics.this_is_exceptional-test_case_per_sec")

	result = mapCounterPathLabel("metrics", "logicaldisk_avg._disk_sec_per_transfer", `RSC Coalesced Packet Bucket 5 (16To31)`)
	assert.Equal(t, result, "metrics.logicaldisk_avg_disk_sec_per_transfer")

}

func TestIsWildcard(t *testing.T) {
	queries := []string{"\\Process(chrome)\\% User Time", "\\Process(chrome#1)\\% User Time", "\\Process(svchost)\\% User Time"}
	instance := "*"
	result := isWildcard(queries, instance)
	assert.True(t, result)
	queries = []string{"\\Process(chrome)\\% User Time"}
	result = isWildcard(queries, instance)
	assert.True(t, result)
	instance = "chrome"
	result = isWildcard(queries, instance)
	assert.False(t, result)
}
