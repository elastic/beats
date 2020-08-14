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

package socket

import "time"

// Config is the configuration specific to the socket MetricSet.
type Config struct {
	ReverseLookup *ReverseLookupConfig `config:"socket.reverse_lookup"`
}

// ReverseLookupConfig contains the configuration that controls the reverse
// DNS lookup behavior.
type ReverseLookupConfig struct {
	Enabled    *bool         `config:"enabled"`
	SuccessTTL time.Duration `config:"success_ttl"`
	FailureTTL time.Duration `config:"failure_ttl"`
}

// IsEnabled returns true if reverse_lookup is defined and 'enabled' is either
// not set or set to true.
func (c *ReverseLookupConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

const (
	defSuccessTTL = 60 * time.Second
	defFailureTTL = 60 * time.Second
)

var defaultConfig = Config{
	ReverseLookup: nil,
}
