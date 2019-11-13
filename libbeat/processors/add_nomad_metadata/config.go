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

package add_nomad_metadata

import (
	"time"
)

// Config for nomad processor.
type Config struct {
	CleanupTimeout time.Duration `config:"cleanup_timeout"`
	Node           string        `config:"node"`
	Region         string        `config:"region"`
	Address        string        `config:"address"`
	SecretID       string        `config:"secret_id"`
	MetaPrefix     string        `config:"meta_prefix"`
}

func defaultConfig() Config {
	return Config{
		CleanupTimeout: 120 * time.Second,
		MetaPrefix:     "logger_",
	}
}
