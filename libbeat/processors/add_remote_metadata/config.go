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

package add_remote_metadata

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type config struct {
	// IgnoreMissing: Ignore errors if event has no matching field.
	IgnoreMissing bool `config:"ignore_missing"`

	// OverwriteKeys allow target_fields to overwrite existing fields.
	OverwriteKeys bool `config:"overwrite_keys"`

	// MatchKeys fields containing the key to lookup.
	MatchKeys []string `config:"match_keys" validate:"required"`

	// Target is the destination root where fields will be added.
	Target string `config:"target"`

	// Fields is the list of fields to add to target.
	Fields []string `config:"include_fields"`

	Provider string `config:"provider" validate:"required"`
}

func defaultConfig() config {
	return config{
		IgnoreMissing: true,
		OverwriteKeys: false,
	}
}

func (pf *config) Validate() error {
	_, ok := providers[pf.Provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", pf.Provider)
	}
	if len(pf.Fields) == 0 && pf.Target == "" {
		return fmt.Errorf("no target field and no source fields specified")
	}
	return nil
}

func (pf *config) getMappings() (mapstr.M, error) {
	m := mapstr.M{}
	prefix := pf.Target
	if len(prefix) > 0 {
		prefix += "."
	}
	for _, src := range pf.Fields {
		dst := prefix + src
		prev, err := m.Put(dst, src)
		if err != nil {
			return nil, fmt.Errorf("failed to set mapping '%v' -> '%v': %w", dst, src, err)
		}
		if prev != nil {
			return nil, fmt.Errorf("field '%v' repeated", src)
		}
	}
	return m.Flatten(), nil
}
