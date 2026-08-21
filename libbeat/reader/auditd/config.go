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

package auditd

// Config stores the configuration for the auditd Parser.
type Config struct {
	// LogErrors, if true, logs parse errors via the parser's logger.
	LogErrors bool `config:"log_errors"`
	// AddErrorKey, if true, adds a parse error to the event under error.message.
	AddErrorKey bool `config:"add_error_key"`
}

// DefaultConfig returns a Config populated with default values.
func DefaultConfig() Config {
	return Config{
		LogErrors:   false,
		AddErrorKey: true,
	}
}
