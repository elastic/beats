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
	"fmt"
	"net/url"
	"strconv"
	"strings"

	rd "github.com/gomodule/redigo/redis"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	mb.BaseMetricSet
	pool *Pool
}

// NewMetricSet creates the base for Redis metricsets.
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	// Unpack additional configuration options.
	config := DefaultConfig()

	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	username, password, dbNumber, err := getUsernamePasswordDBNumber(base.HostData())
	if err != nil {
		return nil, fmt.Errorf("failed to parse username, password and dbNumber from URI: %w", err)
	}

	if config.TLS.IsEnabled() {
		tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
		if err != nil {
			return nil, fmt.Errorf("could not load provided TLS configuration: %w", err)
		}
		config.UseTLSConfig = tlsConfig.ToConfig()
	}

	return &MetricSet{
		BaseMetricSet: base,
		pool: CreatePool(
			base.Host(),
			username,
			password,
			dbNumber,
			&config,
			base.Module().Config().Timeout,
		),
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

// getUserPasswordDBNumber parses username, password and dbNumber from URI or else default
// is used (mentioned in config).
//
// As per security consideration RFC-2396: Uniform Resource Identifiers (URI): Generic Syntax
// https://www.rfc-editor.org/rfc/rfc2396.html#section-7
//
// """
// It is clearly unwise to use a URL that contains a password which is
// intended to be secret. In particular, the use of a password within
// the 'userinfo' component of a URL is strongly disrecommended except
// in those rare cases where the 'password' parameter is intended to be
// public.
// """
//
// In some environments, this is safe but not all. We shouldn't ideally take
// username and password from URI's userinfo or query parameters.
func getUsernamePasswordDBNumber(hostData mb.HostData) (string, string, int, error) {
	// If there are more than one place specified username/password/db-number, use username/password/db-number in query
	uriParsed, err := url.Parse(hostData.URI)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to parse URL '%s': %w", hostData.URI, err)
	}

	// get db-number from URI if it exists
	database := 0
	if uriParsed.Path != "" && strings.HasPrefix(uriParsed.Path, "/") {
		db := strings.TrimPrefix(uriParsed.Path, "/")
		if db != "" {
			database, err = strconv.Atoi(db)
			if err != nil {
				return "", "", 0, fmt.Errorf("redis database in url should be an integer, found: %s: %w", db, err)
			}
		}
	}

	// get username and password from query and also check db-number
	password := hostData.Password
	username := hostData.User
	if uriParsed.RawQuery != "" {
		queryParsed, err := url.ParseQuery(uriParsed.RawQuery)
		if err != nil {
			return "", "", 0, fmt.Errorf("failed to parse query string in '%s': %w", hostData.URI, err)
		}

		usr := queryParsed.Get("username")
		if usr != "" {
			username = usr
		}

		pw := queryParsed.Get("password")
		if pw != "" {
			password = pw
		}

		db := queryParsed.Get("db")
		if db != "" {
			database, err = strconv.Atoi(db)
			if err != nil {
				return "", "", 0, fmt.Errorf("redis database in query should be an integer, found: %s: %w", db, err)
			}
		}
	}

	return username, password, database, nil
}
