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
			"load_error":              c.Int("load_error"),
			"load_success":            c.Int("load_success"),
			"num_keys":                c.Int("num_keys"),
			"override_dir_exists":     c.Int("override_dir_exists"),
			"override_dir_not_exists": c.Int("override_dir_not_exists"),
			"admin_overrides_active":  c.Int("admin_overrides_active", s.Optional),
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
			"live":                           c.Int("live"),
			"memory_allocated":               c.Int("memory_allocated"),
			"memory_heap_size":               c.Int("memory_heap_size"),
			"parent_connections":             c.Int("parent_connections"),
			"total_connections":              c.Int("total_connections"),
			"uptime":                         c.Int("uptime"),
			"version":                        c.Int("version"),
			"watchdog_mega_miss":             c.Int("watchdog_mega_miss", s.Optional),
			"watchdog_miss":                  c.Int("watchdog_miss", s.Optional),
			"hot_restart_epoch":              c.Int("hot_restart_epoch", s.Optional),
		},
		"http2": s.Object{
			"header_overflow":        c.Int("header_overflow", s.Optional),
			"headers_cb_no_stream":   c.Int("headers_cb_no_stream", s.Optional),
			"rx_messaging_error":     c.Int("rx_messaging_error", s.Optional),
			"rx_reset":               c.Int("rx_reset", s.Optional),
			"too_many_header_frames": c.Int("too_many_header_frames", s.Optional),
			"trailers":               c.Int("trailers", s.Optional),
			"tx_reset":               c.Int("tx_reset", s.Optional),
		},
	}
)
var reStats *regexp.Regexp = regexp.MustCompile(`cluster_manager.*|filesystem.*|runtime.*|listener_manager.*|stats.*|server.*|http2\..*`)

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
