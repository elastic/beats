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

package info

import (
	"fmt"
	"strconv"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/redis"
)

var hostParser = parse.URLHostParserBuilder{DefaultScheme: "redis"}.Build()

func init() {
	mb.Registry.MustAddMetricSet("redis", "info", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	*redis.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := redis.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("failed to create 'info' metricset: %w", err)
	}
	return &MetricSet{ms}, nil
}

// Fetch fetches metrics from Redis by issuing the INFO command.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	conn := m.Connection()
	defer func() {
		if err := conn.Close(); err != nil {
			m.Logger().Debug(fmt.Errorf("failed to release connection: %w", err))
		}
	}()

	// Fetch all INFO.
	info, err := redis.FetchRedisInfo("all", conn)
	if err != nil {
		return fmt.Errorf("failed to fetch redis info: %w", err)
	}

	// In 5.0 some fields are renamed, maintain both names, old ones will be deprecated
	renamings := []struct {
		old, new string
	}{
		{"client_longest_output_list", "client_recent_max_output_buffer"},
		{"client_biggest_input_buf", "client_recent_max_input_buffer"},
	}
	for _, r := range renamings {
		if v, ok := info[r.old]; ok {
			info[r.new] = v
			delete(info, r.old)
		}
	}

	slowLogLength, err := redis.FetchSlowLogLength(conn)
	if err != nil {
		return fmt.Errorf("failed to fetch slow log length: %w", err)

	}
	info["slowlog_len"] = strconv.FormatInt(slowLogLength, 10)

	m.Logger().Debugf("Redis INFO from %s: %+v", m.Host(), info)
	eventMapping(r, info)
	return nil
}
