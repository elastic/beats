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
// under the License

package server

import (
	"regexp"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"cluster_manager": s.Object{
			"active_clusters":  c.Int("active_clusters"),
			"cluster_added":    c.Int("cluster_added"),
			"cluster_modified": c.Int("cluster_modified"),
			"cluster_removed":  c.Int("cluster_removed"),
			"warming_clusters": c.Int("warming_clusters"),
		},
		"filesystem": s.Object{
			"flushed_by_timer":     c.Int("flushed_by_timer"),
			"reopen_failed":        c.Int("reopen_failed"),
			"write_buffered":       c.Int("write_buffered"),
			"write_completed":      c.Int("write_completed"),
			"write_total_buffered": c.Int("write_total_buffered"),
		},
		"runtime": s.Object{
			"admin_overrides_active":  c.Int("admin_overrides_active"),
			"load_error":              c.Int("load_error"),
			"load_success":            c.Int("load_success"),
			"num_keys":                c.Int("num_keys"),
			"override_dir_exists":     c.Int("override_dir_exists"),
			"override_dir_not_exists": c.Int("override_dir_not_exists"),
		},
		"listener_manager": s.Object{
			"listener_added":           c.Int("listener_added"),
			"listener_create_failure":  c.Int("listener_create_failure"),
			"listener_create_success":  c.Int("listener_create_success"),
			"listener_modified":        c.Int("listener_modified"),
			"listener_removed":         c.Int("listener_removed"),
			"total_listeners_active":   c.Int("total_listeners_active"),
			"total_listeners_draining": c.Int("total_listeners_draining"),
			"total_listeners_warming":  c.Int("total_listeners_warming"),
		},
		"stats": s.Object{
			"overflow": c.Int("overflow"),
		},
		"server": s.Object{
			"days_until_first_cert_expiring": c.Int("days_until_first_cert_expiring"),
			"live":               c.Int("live"),
			"memory_allocated":   c.Int("memory_allocated"),
			"memory_heap_size":   c.Int("memory_heap_size"),
			"parent_connections": c.Int("parent_connections"),
			"total_connections":  c.Int("total_connections"),
			"uptime":             c.Int("uptime"),
			"version":            c.Int("version"),
			"watchdog_mega_miss": c.Int("watchdog_mega_miss"),
			"watchdog_miss":      c.Int("watchdog_miss"),
		},
		"listener": s.Object{
			"admin": s.Object{
				"downstream_cx_active":  c.Int("downstream_cx_active"),
				"downstream_cx_destroy": c.Int("downstream_cx_destroy"),
				"downstream_cx_total":   c.Int("downstream_cx_total"),
				"http": s.Object{
					"admin": s.Object{
						"downstream_rq_1xx": c.Int("downstream_rq_1xx"),
						"downstream_rq_2xx": c.Int("downstream_rq_2xx"),
						"downstream_rq_3xx": c.Int("downstream_rq_3xx"),
						"downstream_rq_4xx": c.Int("downstream_rq_4xx"),
						"downstream_rq_5xx": c.Int("downstream_rq_5xx"),
					},
				},
			},
		},
		"http": s.Object{
			"admin": s.Object{
				"downstream_cx_active":                          c.Int("downstream_cx_active"),
				"downstream_cx_destroy":                         c.Int("downstream_cx_destroy"),
				"downstream_cx_destroy_active_rq":               c.Int("downstream_cx_destroy_active_rq"),
				"downstream_cx_destroy_local":                   c.Int("downstream_cx_destroy_local"),
				"downstream_cx_destroy_local_active_rq":         c.Int("downstream_cx_destroy_local_active_rq"),
				"downstream_cx_destroy_remote":                  c.Int("downstream_cx_destroy_remote"),
				"downstream_cx_destroy_remote_active_rq":        c.Int("downstream_cx_destroy_remote_active_rq"),
				"downstream_cx_drain_close":                     c.Int("downstream_cx_drain_close"),
				"downstream_cx_http1_active":                    c.Int("downstream_cx_http1_active"),
				"downstream_cx_http1_total":                     c.Int("downstream_cx_http1_total"),
				"downstream_cx_http2_active":                    c.Int("downstream_cx_http2_active"),
				"downstream_cx_http2_total":                     c.Int("downstream_cx_http2_total"),
				"downstream_cx_idle_timeout":                    c.Int("downstream_cx_idle_timeout"),
				"downstream_cx_protocol_error":                  c.Int("downstream_cx_protocol_error"),
				"downstream_cx_rx_bytes_buffered":               c.Int("downstream_cx_rx_bytes_buffered"),
				"downstream_cx_rx_bytes_total":                  c.Int("downstream_cx_rx_bytes_total"),
				"downstream_cx_ssl_active":                      c.Int("downstream_cx_ssl_active"),
				"downstream_cx_ssl_total":                       c.Int("downstream_cx_ssl_total"),
				"downstream_cx_total":                           c.Int("downstream_cx_total"),
				"downstream_cx_tx_bytes_buffered":               c.Int("downstream_cx_tx_bytes_buffered"),
				"downstream_cx_tx_bytes_total":                  c.Int("downstream_cx_tx_bytes_total"),
				"downstream_cx_websocket_active":                c.Int("downstream_cx_websocket_active"),
				"downstream_cx_websocket_total":                 c.Int("downstream_cx_websocket_total"),
				"downstream_flow_control_paused_reading_total":  c.Int("downstream_flow_control_paused_reading_total"),
				"downstream_flow_control_resumed_reading_total": c.Int("downstream_flow_control_resumed_reading_total"),
				"downstream_rq_1xx":                             c.Int("downstream_rq_1xx"),
				"downstream_rq_2xx":                             c.Int("downstream_rq_2xx"),
				"downstream_rq_3xx":                             c.Int("downstream_rq_3xx"),
				"downstream_rq_4xx":                             c.Int("downstream_rq_4xx"),
				"downstream_rq_5xx":                             c.Int("downstream_rq_5xx"),
				"downstream_rq_active":                          c.Int("downstream_rq_active"),
				"downstream_rq_http1_total":                     c.Int("downstream_rq_http1_total"),
				"downstream_rq_http2_total":                     c.Int("downstream_rq_http2_total"),
				"downstream_rq_non_relative_path":               c.Int("downstream_rq_non_relative_path"),
				"downstream_rq_response_before_rq_complete":     c.Int("downstream_rq_response_before_rq_complete"),
				"downstream_rq_rx_reset":                        c.Int("downstream_rq_rx_reset"),
				"downstream_rq_too_large":                       c.Int("downstream_rq_too_large"),
				"downstream_rq_total":                           c.Int("downstream_rq_total"),
				"downstream_rq_tx_reset":                        c.Int("downstream_rq_tx_reset"),
				"downstream_rq_ws_on_non_ws_route":              c.Int("downstream_rq_ws_on_non_ws_route"),
				"rs_too_large":                                  c.Int("rs_too_large"),
			},
			"async-client": s.Object{
				"no_cluster":         c.Int("no_cluster"),
				"no_route":           c.Int("no_route"),
				"rq_direct_response": c.Int("rq_direct_response"),
				"rq_redirect":        c.Int("rq_redirect"),
				"rq_total":           c.Int("rq_total"),
			},
		},
	}
)
var reStats *regexp.Regexp = regexp.MustCompile(`cluster_manager.*|filesystem.*|runtime.*|listener_manager.*|stats.*|server.*|listener.*|http.*`)

func eventMapping(response []byte) (common.MapStr, error) {
	data := map[string]interface{}{}
	var events common.MapStr
	var err error

	data = findStats(data, response)
	events, err = schema.Apply(data)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func findStats(data common.MapStr, response []byte) common.MapStr {
	matches := reStats.FindAllString(string(response), -1)
	for i := 0; i < len(matches); i++ {
		entries := strings.Split(matches[i], ": ")
		if len(entries) == 2 {
			temp := strings.Split(entries[0], ".")
			data[temp[len(temp)-1]] = entries[1]
		}
	}
	return data
}
