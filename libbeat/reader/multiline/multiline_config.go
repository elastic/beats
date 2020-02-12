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

package multiline

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/match"
)

// Config holds the options of multiline readers.
type Config struct {
	Negate       bool           `config:"negate"`
	Match        string         `config:"match" validate:"required"`
	MaxLines     *int           `config:"max_lines"`
	Pattern      *match.Matcher `config:"pattern" validate:"required"`
	Timeout      *time.Duration `config:"timeout" validate:"positive"`
	FlushPattern *match.Matcher `config:"flush_pattern"`
}

// Validate validates the Config option for multiline reader.
func (c *Config) Validate() error {
	if c.Match != "after" && c.Match != "before" {
		return fmt.Errorf("unknown matcher type: %s", c.Match)
	}
	return nil
}
