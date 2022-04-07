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

//go:build integration
// +build integration

package redis

import (
	"strings"
	"testing"

	rd "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/tests/compose"
	_ "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

func TestFetchRedisInfo(t *testing.T) {
	service := compose.EnsureUp(t, "redis")
	host := service.Host()

	conn, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer conn.Close()

	t.Run("default info", func(t *testing.T) {
		info, err := FetchRedisInfo("default", conn)
		require.NoError(t, err)

		_, ok := info["redis_version"]
		assert.True(t, ok, "redis_version should be in redis info")
	})

	t.Run("keyspace info", func(t *testing.T) {
		conn.Do("FLUSHALL")
		conn.Do("SET", "foo", "bar")

		info, err := FetchRedisInfo("keyspace", conn)
		require.NoError(t, err)

		dbFound := false
		for k := range info {
			if strings.HasPrefix(k, "db") {
				dbFound = true
				break
			}
		}
		assert.True(t, dbFound, "there should be keyspaces in redis info")
	})
}

func TestFetchKeys(t *testing.T) {
	service := compose.EnsureUp(t, "redis")
	host := service.Host()

	conn, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer conn.Close()

	conn.Do("FLUSHALL")
	conn.Do("SET", "foo", "bar")
	conn.Do("LPUSH", "foo-list", "42")

	k, err := FetchKeys(conn, "notexist", 0)
	assert.NoError(t, err)
	assert.Empty(t, k)

	k, err = FetchKeys(conn, "foo", 0)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"foo"}, k)

	k, err = FetchKeys(conn, "foo*", 0)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"foo", "foo-list"}, k)
}

func TestFetchKeyInfo(t *testing.T) {
	service := compose.EnsureUp(t, "redis")
	host := service.Host()

	conn, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer conn.Close()

	conn.Do("FLUSHALL")

	cases := []struct {
		Title    string
		Key      string
		Command  string
		Value    []interface{}
		Expire   uint
		Expected map[string]interface{}
	}{
		{
			Title:   "plain string",
			Key:     "string-key",
			Command: "SET",
			Value:   []interface{}{"foo"},
			Expected: map[string]interface{}{
				"name":   "string-key",
				"type":   "string",
				"length": int64(3),
				"expire": map[string]interface{}{
					"ttl": int64(-1),
				},
			},
		},
		{
			Title:   "plain string with TTL",
			Key:     "string-key",
			Command: "SET",
			Value:   []interface{}{"foo"},
			Expire:  60,
			Expected: map[string]interface{}{
				"name":   "string-key",
				"type":   "string",
				"length": int64(3),
				"expire": map[string]interface{}{
					"ttl": int64(60),
				},
			},
		},
		{
			Title:   "list",
			Key:     "list-key",
			Command: "LPUSH",
			Value:   []interface{}{"foo", "bar"},
			Expected: map[string]interface{}{
				"name":   "list-key",
				"type":   "list",
				"length": int64(2),
				"expire": map[string]interface{}{
					"ttl": int64(-1),
				},
			},
		},
		{
			Title:   "set",
			Key:     "set-key",
			Command: "SADD",
			Value:   []interface{}{"foo", "bar"},
			Expected: map[string]interface{}{
				"name":   "set-key",
				"type":   "set",
				"length": int64(2),
				"expire": map[string]interface{}{
					"ttl": int64(-1),
				},
			},
		},
		{
			Title:   "sorted set",
			Key:     "sorted-set-key",
			Command: "ZADD",
			Value:   []interface{}{1, "foo", 2, "bar"},
			Expected: map[string]interface{}{
				"name":   "sorted-set-key",
				"type":   "zset",
				"length": int64(2),
				"expire": map[string]interface{}{
					"ttl": int64(-1),
				},
			},
		},
		{
			Title:   "hash",
			Key:     "hash-key",
			Command: "HSET",
			Value:   []interface{}{"foo", "bar"},
			Expected: map[string]interface{}{
				"name":   "hash-key",
				"type":   "hash",
				"length": int64(1),
				"expire": map[string]interface{}{
					"ttl": int64(-1),
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Title, func(t *testing.T) {
			args := append([]interface{}{c.Key}, c.Value...)
			conn.Do(c.Command, args...)
			defer conn.Do("DEL", c.Key)
			if c.Expire > 0 {
				conn.Do("EXPIRE", c.Key, c.Expire)
			}

			info, err := FetchKeyInfo(conn, c.Key)
			require.NoError(t, err)
			require.Equal(t, c.Expected, info)
		})
	}
}
