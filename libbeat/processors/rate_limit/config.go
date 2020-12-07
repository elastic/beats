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

package rate_limit

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors/rate_limit/algorithm"
)

// Config for rate limit processor.
type Config struct {
	Limit     algorithm.Rate         `config:"limit" validate:"required"`
	Fields    []string               `config:"fields"`
	Algorithm common.ConfigNamespace `config:"algorithm"`
}

func defaultConfig() (*Config, error) {
	cfg, err := common.NewConfigFrom(map[string]interface{}{
		"algorithm": map[string]interface{}{
			"token_bucket": map[string]interface{}{
				"burst_multiplier": 1.0,
			},
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "could not parse default configuration")
	}

	var config Config
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "could not unpack default configuration")
	}

	config.Fields = make([]string, 0)

	return &config, nil
}
