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

package processors

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// PluginConfig represents the list of processors given in beat configuration.
type PluginConfig []*config.C

// MandatoryExportedFields are fields that should be always exported
var MandatoryExportedFields = []string{"type"}

// NewPluginConfigFromList creates a PluginConfig from a list of raw processor config objects
func NewPluginConfigFromList(raw []mapstr.M) (PluginConfig, error) {
	processors := make([]*config.C, len(raw))
	for i := 0; i < len(raw); i++ {
		cfg, err := config.NewConfigFrom(raw[i])
		if err != nil {
			return nil, fmt.Errorf("error creating processor config: %w", err)
		}
		processors[i] = cfg
	}

	return processors, nil
}
