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

package mtest

import (
	"net"
	"os"
)

// GetEnvHost returns host for Kibana
func GetEnvHost() string {
	host := os.Getenv("KIBANA_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort returns port for Kibana
func GetEnvPort() string {
	port := os.Getenv("KIBANA_PORT")

	if len(port) == 0 {
		port = "5601"
	}
	return port
}

// GetConfig returns config for kibana module
func GetConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kibana",
		"metricsets": []string{metricset},
		"hosts":      []string{net.JoinHostPort(GetEnvHost(), GetEnvPort())},
	}
}
