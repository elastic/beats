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

	"github.com/menderesk/beats/v7/libbeat/reader"
)

// New creates a new multi-line reader combining stream of
// line events into stream of multi-line events.
func New(
	r reader.Reader,
	separator string,
	maxBytes int,
	config *Config,
) (reader.Reader, error) {
	switch config.Type {
	case patternMode:
		return newMultilinePatternReader(r, separator, maxBytes, config)
	case countMode:
		return newMultilineCountReader(r, separator, maxBytes, config)
	case whilePatternMode:
		return newMultilineWhilePatternReader(r, separator, maxBytes, config)
	default:
		return nil, fmt.Errorf("unknown multiline type %d", config.Type)
	}
}
