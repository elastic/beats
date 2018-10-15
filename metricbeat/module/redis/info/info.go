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
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/redis"

	rd "github.com/garyburd/redigo/redis"
)

var (
	debugf = logp.MakeDebug("redis-info")
)

func init() {
	mb.Registry.MustAddMetricSet("redis", "info", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	mb.BaseMetricSet
	pool *rd.Pool
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		IdleTimeout time.Duration `config:"idle_timeout"`
		Network     string        `config:"network"`
		MaxConn     int           `config:"maxconn" validate:"min=1"`
		Password    string        `config:"password"`
	}{
		Network:  "tcp",
		MaxConn:  10,
		Password: "",
	}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		pool: redis.CreatePool(base.Host(), config.Password, config.Network,
			config.MaxConn, config.IdleTimeout, base.Module().Config().Timeout),
	}, nil
}

// Fetch fetches metrics from Redis by issuing the INFO command.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	// Fetch default INFO.
	info, err := redis.FetchRedisInfo("default", m.pool.Get())
	if err != nil {
		return nil, err
	}

	// In 5.0 some fields are renamed, maintain both names, old ones will be deprecated
	renamings := []struct {
		old, new string
	}{
		{"client_longest_output_list", "client_recent_max_output_buffer"},
		{"client_biggest_input_buf", "client_recent_max_input_buffer"},
	}
	for _, r := range renamings {
		if v, ok := info[r.new]; ok {
			info[r.old] = v
		} else {
			info[r.new] = info[r.old]
		}
	}

	slowLogLength, err := redis.FetchSlowLogLength(m.pool.Get())
	if err != nil {
		return nil, err
	}
	info["slowlog_len"] = strconv.FormatInt(slowLogLength, 10)

	debugf("Redis INFO from %s: %+v", m.Host(), info)
	return eventMapping(info), nil
}
