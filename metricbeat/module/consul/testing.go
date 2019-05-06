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

package consul

import (
	"fmt"
	"os"
)

//GetConfig returns a config object specific for a Consul module and a provided Metricset in 'ms'
func GetConfig(ms []string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "consul",
		"metricsets": ms,
		"hosts":      []string{fmt.Sprintf("%s:%s", EnvOr("CONSUL_HOST", "localhost"), EnvOr("CONSUL_PORT", "8500"))},
	}
}

// EnvOr returns the value of the specified environment variable if it is
// non-empty. Otherwise it return def.
func EnvOr(name, def string) string {
	s := os.Getenv(name)
	if s == "" {
		return def
	}
	return s
}
