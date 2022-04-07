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
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v8/libbeat/common/match"
)

type multilineType uint8

const (
	patternMode multilineType = iota
	countMode
	whilePatternMode

	patternStr      = "pattern"
	countStr        = "count"
	whilePatternStr = "while_pattern"
)

var (
	multilineTypes = map[string]multilineType{
		patternStr:      patternMode,
		countStr:        countMode,
		whilePatternStr: whilePatternMode,
	}

	ErrMissingPattern = errors.New("multiline.pattern cannot be empty when pattern based matching is selected")
	ErrMissingCount   = errors.New("multiline.count cannot be empty when count based aggregation is selected")
)

// Config holds the options of multiline readers.
type Config struct {
	Type multilineType `config:"type"`

	Negate       bool           `config:"negate"`
	Match        string         `config:"match"`
	MaxLines     *int           `config:"max_lines"`
	Pattern      *match.Matcher `config:"pattern"`
	Timeout      *time.Duration `config:"timeout" validate:"positive"`
	FlushPattern *match.Matcher `config:"flush_pattern"`

	LinesCount  int  `config:"count_lines" validate:"positive"`
	SkipNewLine bool `config:"skip_newline"`
}

// Validate validates the Config option for multiline reader.
func (c *Config) Validate() error {
	if c.Type == patternMode {
		if c.Match != "after" && c.Match != "before" {
			return fmt.Errorf("unknown matcher type: %s", c.Match)
		}
		if c.Pattern == nil {
			return ErrMissingPattern
		}
	} else if c.Type == countMode {
		if c.LinesCount == 0 {
			return ErrMissingCount
		}
	} else if c.Type == whilePatternMode {
		if c.Pattern == nil {
			return ErrMissingPattern
		}
	} else {
		return fmt.Errorf("unknown multiline type %d", c.Type)
	}
	return nil
}

// Unpack selects the approriate aggregation method for creating multiline events.
// If it is not configured pattern matching is chosen.
func (m *multilineType) Unpack(value string) error {
	if value == "" {
		*m = patternMode
		return nil
	}

	s, ok := multilineTypes[value]
	if !ok {
		return fmt.Errorf("unknown multiline type: %s", value)
	}
	*m = s
	return nil
}
