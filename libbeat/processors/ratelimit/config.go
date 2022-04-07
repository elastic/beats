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

package ratelimit

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
)

// config for rate limit processor.
type config struct {
	Limit     rate                   `config:"limit" validate:"required"`
	Fields    []string               `config:"fields"`
	Algorithm common.ConfigNamespace `config:"algorithm"`
}

func (c *config) setDefaults() error {
	if c.Algorithm.Name() == "" {
		cfg, err := common.NewConfigFrom(map[string]interface{}{
			"token_bucket": map[string]interface{}{},
		})

		if err != nil {
			return errors.Wrap(err, "could not parse default configuration")
		}

		c.Algorithm.Unpack(cfg)
	}

	return nil
}
