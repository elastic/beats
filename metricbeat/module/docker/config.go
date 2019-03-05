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

package docker

type Config struct {
	TLS   *TLSConfig `config:"ssl"`
	DeDot bool       `config:"labels.dedot"`
}

// DefaultConfig returns default module config
func DefaultConfig() Config {
	return Config{
		DeDot: true,
	}
}

type TLSConfig struct {
	Enabled     *bool  `config:"enabled"`
	CA          string `config:"certificate_authority"`
	Certificate string `config:"certificate"`
	Key         string `config:"key"`
}

func (c *TLSConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
