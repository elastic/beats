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
	"os"
)

// Helper functions for testing used in the redis metricsets

// GetRedisEnvHost returns the hostname of the Redis server to use for testing.
// It reads the value from the REDIS_HOST environment variable and returns
// 127.0.0.1 if it is not set.
func GetRedisEnvHost() string {
	host := os.Getenv("REDIS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetRedisEnvPort returns the port of the Redis server to use for testing.
// It reads the value from the REDIS_PORT environment variable and returns
// 6379 if it is not set.
func GetRedisEnvPort() string {
	port := os.Getenv("REDIS_PORT")

	if len(port) == 0 {
		port = "6379"
	}
	return port
}
