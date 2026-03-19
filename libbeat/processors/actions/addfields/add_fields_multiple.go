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

package addfields

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func CreateAddFieldsMultiple(c *conf.C, _ *logp.Logger) (beat.Processor, error) {
	var entries []struct {
		Fields mapstr.M `config:"fields" validate:"required"`
		Target *string  `config:"target"`
	}
	if err := c.Unpack(&entries); err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields_multiple configuration: %w", err)
	}

	merged := mapstr.M{}
	for _, e := range entries {
		wrapped := e.Fields
		target := optTarget(e.Target, FieldsKey)
		if target != "" {
			wrapped = mapstr.M{target: wrapped}
		}
		merged.DeepUpdate(wrapped)
	}

	return &addFields{fields: merged, shared: true, overwrite: true}, nil
}
