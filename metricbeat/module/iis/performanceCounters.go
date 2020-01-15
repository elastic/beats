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
	website_counters = map[string]string{
		"bytes_sent_per_sec":       "\\Web Service(*)\\Bytes Sent/sec",
		"total_bytes_sent_per_sec": "\\Web Service(*)\\Total Bytes Sent",
		"bytes_recv_per_sec":       "\\Web Service(*)\\Bytes Received/sec",
		"total_bytes_recv_per_sec": "\\Web Service(*)\\Total Bytes Received",

		//\Web Service(*)\Total Files Sent
		//\Web Service(*)\Files Sent/sec
		//\Web Service(*)\Total Files Received
		//\Web Service(*)\Files Received/sec
		//\Web Service(*)\Current Connections
		//\Web Service(*)\Maximum Connections
		//\Web Service(*)\Total Connection Attempts (all instances)
		//\Web Service(*)\Total Get Requests
		//\Web Service(*)\Get Requests/sec
		//\Web Service(*)\Total Post Requests
		//\Web Service(*)\Post Requests/sec
	}
	webserver_counters = map[string]string{
		"total_bytes_sent_per_sec": "\\Web Service(_Total)\\Total Bytes Sent",
		"total_bytes_recv_per_sec": "\\Web Service(_Total)\\Total Bytes Received",
		//\Web Service(*)\Total Files Sent
		//\Web Service(*)\Files Sent/sec
		//\Web Service(*)\Total Files Received
		//\Web Service(*)\Files Received/sec
		//\Web Service(*)\Current Connections
		//\Web Service(*)\Maximum Connections
		//\Web Service(*)\Total Connection Attempts (all instances)
		//\Web Service(*)\Total Get Requests
		//\Web Service(*)\Get Requests/sec
		//\Web Service(*)\Total Post Requests
		//\Web Service(*)\Post Requests/sec

		//cache
		//"cache": {
		//"file_cache_count": "2",
		//"file_cache_memory_usage": "699",
		//"file_cache_hits": "18506471",
		//"file_cache_misses": "46266060",
		//"total_files_cached": "10",
		//"output_cache_count": "0",
		//"output_cache_memory_usage": "0",
		//"output_cache_hits": "0",
		//"output_cache_misses": "18506478",
		//"uri_cache_count": "2",
		//"uri_cache_hits": "18506452",
		//"uri_cache_misses": "26",
		//"total_uris_cached": "13"
		//}

	}
)

type PerformanceCounter struct {
	InstanceLabel    string
	MeasurementLabel string
	Path             string
	Format           string
}

func GetPerfCounters(metricset string) []PerformanceCounter {
	var counters []PerformanceCounter
	switch metricset {
	case "website":
		for k, v := range website_counters {
			counter := PerformanceCounter{
				InstanceLabel:    "name",
				MeasurementLabel: k,
				Path:             v,
				Format:           "float",
			}
			counters = append(counters, counter)
		}
	case "webserver":
		for k, v := range webserver_counters {
			counter := PerformanceCounter{
				InstanceLabel:    "",
				MeasurementLabel: k,
				Path:             v,
				Format:           "float",
			}
			counters = append(counters, counter)
		}

	}
	return counters
}
