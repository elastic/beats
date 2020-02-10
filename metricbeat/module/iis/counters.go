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

package iis

var (
	WebsiteCounters = map[string]string{
		"network.total_bytes_received":      "\\Web Service(*)\\Total Bytes Received",
		"network.total_bytes_sent":          "\\Web Service(*)\\Total Bytes Sent",
		"network.bytes_sent_per_sec":        "\\Web Service(*)\\Bytes Sent/sec",
		"network.bytes_received_per_sec":    "\\Web Service(*)\\Bytes Received/sec",
		"network.current_connections":       "\\Web Service(*)\\Current Connections",
		"network.maximum_connections":       "\\Web Service(*)\\Maximum Connections",
		"network.total_connection_attempts": "\\Web Service(*)\\Total Connection Attempts (all instances)",
		"network.total_get_requests":        "\\Web Service(*)\\Total Get Requests",
		"network.get_requests_per_sec":      "\\Web Service(*)\\Get Requests/sec",
		"network.total_post_requests":       "\\Web Service(*)\\Total Post Requests",
		"network.post_requests_per_sec":     "\\Web Service(*)\\Post Requests/sec",
		"network.total_delete_requests":     "\\Web Service(*)\\Total Delete Requests",
		"network.delete_requests_per_sec":   "\\Web Service(*)\\Delete Requests/sec",
		"network.service_uptime":            "\\Web Service(*)\\Service Uptime",
	}
	AppPoolCounters = map[string]string{
		"worker_process_id":                    "\\Process(w3wp*)\\ID Process",
		"process.cpu_usage_perc":               "\\Process(w3wp*)\\% Processor Time",
		"process.handle_count":                 "\\Process(w3wp*)\\Handle Count",
		"process.thread_count":                 "\\Process(w3wp*)\\Thread Count",
		"process.working_set":                  "\\Process(w3wp*)\\Working Set",
		"process.private_byte":                 "\\Process(w3wp*)\\Private Bytes",
		"process.virtual_bytes":                "\\Process(w3wp*)\\Virtual Bytes",
		"process.page_faults_per_sec":          "\\Process(w3wp*)\\Page Faults/sec",
		"process.io_read_operations_per_sec":   "\\Process(w3wp*)\\IO Read Operations/sec",
		"process.io_write_operations_per_sec":  "\\Process(w3wp*)\\IO Write Operations/sec",
		"net_clr.total_exceptions_thrown":      "\\.NET CLR Exceptions(w3wp*)\\# of Exceps Thrown",
		"net_clr.exceptions_thrown_per_sec":    "\\.NET CLR Exceptions(w3wp*)\\# of Exceps Thrown / sec",
		"net_clr.filters_per_sec":              "\\.NET CLR Exceptions(w3wp*)\\# of Filters / sec",
		"net_clr.finallys_per_sec":             "\\.NET CLR Exceptions(w3wp*)\\# of Finallys / sec",
		"net_clr.throw_to_catch_depth_per_sec": "\\.NET CLR Exceptions(w3wp*)\\Throw To Catch Depth / sec",
		//"asp_net_applications.request_bytes_total": "\\ASP.NET Applications(w3wp*)\\Request Bytes In Total",
	}
)
