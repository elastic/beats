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

/*
Package redis contains shared Redis functionality for the metric sets
*/
package redis

import (
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	rd "github.com/garyburd/redigo/redis"
)

// ParseRedisInfo parses the string returned by the INFO command
// Every line is split up into key and value
func ParseRedisInfo(info string) map[string]string {
	// Feed every line into
	result := strings.Split(info, "\r\n")

	// Load redis info values into array
	values := map[string]string{}

	for _, value := range result {
		// Values are separated by :
		parts := ParseRedisLine(value, ":")
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}
	return values
}

// ParseRedisLine parses a single line returned by INFO
func ParseRedisLine(s string, delimiter string) []string {
	return strings.Split(s, delimiter)
}

// FetchRedisInfo returns a map of requested stats.
func FetchRedisInfo(stat string, c rd.Conn) (map[string]string, error) {
	defer c.Close()

	out, err := rd.String(c.Do("INFO", stat))
	if err != nil {
		logp.Err("Error retrieving INFO stats: %v", err)
		return nil, err
	}
	return ParseRedisInfo(out), nil
}

// FetchSlowLogLength returns count of slow operations
func FetchSlowLogLength(c rd.Conn) (int64, error) {
	count, err := rd.Int64(c.Do("SLOWLOG", "len"))
	if err != nil {
		logp.Err("Error retrieving slowlog len: %v", err)
		return 0, err
	}

	return count, nil
}

// CreatePool creates a redis connection pool
func CreatePool(
	host, password, network string,
	maxConn int,
	idleTimeout, connTimeout time.Duration,
) *rd.Pool {
	return &rd.Pool{
		MaxIdle:     maxConn,
		IdleTimeout: idleTimeout,
		Dial: func() (rd.Conn, error) {
			c, err := rd.Dial(network, host,
				rd.DialConnectTimeout(connTimeout),
				rd.DialReadTimeout(connTimeout),
				rd.DialWriteTimeout(connTimeout))
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}
