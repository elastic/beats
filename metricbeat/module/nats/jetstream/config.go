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

package jetstream

type ModuleConfig struct {
	Jetstream MetricsetConfig `config:"jetstream"`
}

type MetricsetConfig struct {
	Account  AccountConfig  `config:"account"`
	Stats    StatsConfig    `config:"stats"`
	Stream   StreamConfig   `config:"stream"`
	Consumer ConsumerConfig `config:"consumer"`
}

type AccountConfig struct {
	Enabled bool     `config:"enabled"`
	Names   []string `config:"names"`
}

type StatsConfig struct {
	Enabled bool `config:"enabled"`
}

type StreamConfig struct {
	Enabled bool     `config:"enabled"`
	Names   []string `config:"names"`
}

type ConsumerConfig struct {
	Enabled bool     `config:"enabled"`
	Names   []string `config:"names"`
}
