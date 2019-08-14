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
	"os"
)

// GetIntegrationConfig generates a base configuration with common values for
// integration tests
func GetIntegrationConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":   "rabbitmq",
		"hosts":    []string{host},
		"username": getTestRabbitMQUsername(),
		"password": getTestRabbitMQPassword(),
	}
}

const (
	rabbitmqDefaultUsername = "guest"
	rabbitmqDefaultPassword = "guest"
)

func getTestRabbitMQUsername() string {
	return getenv("RABBITMQ_USERNAME", rabbitmqDefaultUsername)
}

func getTestRabbitMQPassword() string {
	return getenv("RABBITMQ_PASSWORD", rabbitmqDefaultPassword)
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}
