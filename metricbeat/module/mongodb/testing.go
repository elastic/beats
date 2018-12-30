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

package mongodb

import "os"

// Helper functions for testing mongodb metricsets.

// GetEnvHost returns the hostname of the Mongodb server to use for testing.
// It reads the value from the MONGODB_HOST environment variable and returns
// 127.0.0.1 if it is not set.
func GetEnvHost() string {
	host := os.Getenv("MONGODB_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort returns the port of the Mongodb server to use for testing.
// It reads the value from the MONGODB_PORT environment variable and returns
// 27017 if it is not set.
func GetEnvPort() string {
	port := os.Getenv("MONGODB_PORT")

	if len(port) == 0 {
		port = "27017"
	}
	return port
}
