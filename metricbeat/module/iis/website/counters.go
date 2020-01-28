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

package website

var (
	webserverCounters = map[string]string{
		"network.total_bytes_received":            "\\Web Service(*)\\Total Bytes Received",
		"network.total_bytes_sent":                "\\Web Service(*)\\Total Bytes Sent",
		"network.bytes_sent_per_sec":              "\\Web Service(*)\\Bytes Sent/sec",
		"network.bytes_received_per_sec":          "\\Web Service(*)\\Bytes Received/sec",
		"network.current_connections":             "\\Web Service(*)\\Current Connections",
		"network.maximum_connections":             "\\Web Service(*)\\Maximum Connections",
		"network.total_connection_attempts":       "\\Web Service(*)\\Total Connection Attempts (all instances)",
		"network.total_get_requests":              "\\Web Service(*)\\Total Get Requests",
		"network.get_requests_per_sec":            "\\Web Service(*)\\Get Requests/sec",
		"network.total_post_requests":             "\\Web Service(*)\\Total Post Requests",
		"network.post_requests_per_sec":           "\\Web Service(*)\\Post Requests/sec",
		"network.total_delete_requests":           "\\Web Service(*)\\Total Delete Requests",
		"network.delete_requests_per_sec":         "\\Web Service(*)\\Delete Requests/sec",
		"network.service_uptime":                  "\\Web Service(*)\\Service Uptime",
		"cache.current_files_cached":              "\\Web Service Cache\\Current Files Cached",
		"cache.total_files_cached":                "\\Web Service Cache\\Total Files Cached",
		"cache.file_cache_hits":                   "\\Web Service Cache\\File Cache Hits",
		"cache.file_cache_misses":                 "\\Web Service Cache\\File Cache Misses",
		"cache.current_file_cache_memory_usage":   "\\Web Service Cache\\Current File Cache Memory Usage",
		"cache.maximum_file_cache_memory_usage":   "\\Web Service Cache\\Maximum File Cache Memory Usage",
		"cache.current_uris_cached":               "\\Web Service Cache\\Current URIs Cached",
		"cache.total_uris_cached":                 "\\Web Service Cache\\Total URIs Cached",
		"cache.uri_cache_hits":                    "\\Web Service Cache\\URI Cache Hits",
		"cache.uri_cache_misses":                  "\\Web Service Cache\\URI Cache Misses",
		"cache.output_cache_current_memory_usage": "\\Web Service Cache\\Output Cache Current Memory Usage",
		"cache.output_cache_current_items":        "\\Web Service Cache\\Output Cache Current Items",
		"cache.output_cache_total_hits":           "\\Web Service Cache\\Output Cache Total Hits",
		"cache.output_cache_total_misses":         "\\Web Service Cache\\Output Cache Total Misses",
	}
	processCounters = map[string]string{
		"process.cpu_usage_perc":              "\\Process(*)\\% Processor Time",
		"process.handle_count":                "\\Process(*)\\Handle Count",
		"process.thread_count":                "\\Process(*)\\Thread Count",
		"process.working_set":                 "\\Process(*)\\Working Set",
		"process.private_byte":                "\\Process(*)\\Private Bytes",
		"process.virtual_bytes":               "\\Process(*)\\Virtual Bytes",
		"process.page_faults_per_sec":         "\\Process(*)\\Page Faults/sec",
		"process.io_read_operations_per_sec":  "\\Process(*)\\IO Read Operations/sec",
		"process.io_write_operations_per_sec": "\\Process(*)\\IO Write Operations/sec",
	}
)
