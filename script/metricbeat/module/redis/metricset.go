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
	"net/url"
	"strconv"
	"strings"
	"time"

	rd "github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
)

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	mb.BaseMetricSet
	pool *Pool
}

// NewMetricSet creates the base for Redis metricsets
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		IdleTimeout time.Duration `config:"idle_timeout"`
		Network     string        `config:"network"`
		MaxConn     int           `config:"maxconn" validate:"min=1"`
	}{
		Network: "tcp",
		MaxConn: 10,
	}

	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configuration")
	}

	password, dbNumber, err := getPasswordDBNumber(base.HostData())
	if err != nil {
		return nil, errors.Wrap(err, "failed to getPasswordDBNumber from URI")
	}

	return &MetricSet{
		BaseMetricSet: base,
		pool: CreatePool(base.Host(), password, config.Network, dbNumber,
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

// OriginalDBNumber returns the originally configured database number, this can be used by
// metricsets that change keyspace to go back to the originally configured one
func (m *MetricSet) OriginalDBNumber() uint {
	return uint(m.pool.DBNumber())
}

func getPasswordDBNumber(hostData mb.HostData) (string, int, error) {
	// If there are more than one place specified password/db-number, use password/db-number in query
	uriParsed, err := url.Parse(hostData.URI)
	if err != nil {
		return "", 0, errors.Wrapf(err, "failed to parse URL '%s'", hostData.URI)
	}

	// get db-number from URI if it exists
	database := 0
	if uriParsed.Path != "" && strings.HasPrefix(uriParsed.Path, "/") {
		db := strings.TrimPrefix(uriParsed.Path, "/")
		if db != "" {
			database, err = strconv.Atoi(db)
			if err != nil {
				return "", 0, errors.Wrapf(err, "redis database in url should be an integer, found: %s", db)
			}
		}
	}

	// get password from query and also check db-number
	password := hostData.Password
	if uriParsed.RawQuery != "" {
		queryParsed, err := url.ParseQuery(uriParsed.RawQuery)
		if err != nil {
			return "", 0, errors.Wrapf(err, "failed to parse query string in '%s'", hostData.URI)
		}

		pw := queryParsed.Get("password")
		if pw != "" {
			password = pw
		}

		db := queryParsed.Get("db")
		if db != "" {
			database, err = strconv.Atoi(db)
			if err != nil {
				return "", 0, errors.Wrapf(err, "redis database in query should be an integer, found: %s", db)
			}
		}
	}
	return password, database, nil
}
