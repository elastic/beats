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

package input

import (
	"time"

	cfg "github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
)

var defaultConfig = inputConfig{
	ScanFrequency: 10 * time.Second,
	Type:          cfg.DefaultType,
}

type inputConfig struct {
	ScanFrequency time.Duration `config:"scan_frequency" validate:"min=0,nonzero"`
	Type          string        `config:"type"`
	InputType     string        `config:"input_type"`
}

func (c *inputConfig) Validate() error {
	if c.InputType != "" {
		cfgwarn.Deprecate("6.0.0", "input_type input config is deprecated. Use type instead.")
		c.Type = c.InputType
	}
	return nil
}
