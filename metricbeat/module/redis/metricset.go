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

package redis

import (
	"time"

	rd "github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	mb.BaseMetricSet
	pool *rd.Pool
}

// NewMetricSet creates the base for Redis metricsets
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
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
		return nil, errors.Wrap(err, "failed to read configuration")
	}

	return &MetricSet{
		BaseMetricSet: base,
		pool: CreatePool(base.Host(), config.Password, config.Network,
			config.MaxConn, config.IdleTimeout, base.Module().Config().Timeout),
	}, nil
}

// Connection returns a redis connection from the pool
func (m *MetricSet) Connection() rd.Conn {
	return m.pool.Get()
}

// Close redis connections
func (m *MetricSet) Close() error {
	return m.pool.Close()
}
