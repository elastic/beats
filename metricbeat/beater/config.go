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

package beater

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Config is the root of the Metricbeat configuration hierarchy.
type Config struct {
	// Modules is a list of module specific configuration data.
	Modules       []*conf.C          `config:"modules"`
	ConfigModules *conf.C            `config:"config.modules"`
	MaxStartDelay time.Duration        `config:"max_start_delay"` // Upper bound on the random startup delay for metricsets (use 0 to disable startup delay).
	Autodiscover  *autodiscover.Config `config:"autodiscover"`
}

var defaultConfig = Config{
	MaxStartDelay: 10 * time.Second,
}
