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

package javascript

import (
	"time"

	"github.com/pkg/errors"
)

// Config defines the Javascript source files to use for the processor.
type Config struct {
	Tag            string                 `config:"tag"`                      // Processor ID for debug and metrics.
	Source         string                 `config:"source"`                   // Inline script to execute.
	File           string                 `config:"file"`                     // Source file.
	Files          []string               `config:"files"`                    // Multiple source files.
	Params         map[string]interface{} `config:"params"`                   // Parameters to pass to script.
	Timeout        time.Duration          `config:"timeout" validate:"min=0"` // Execution timeout.
	TagOnException string                 `config:"tag_on_exception"`         // Tag to add to events when an exception happens.
}

// Validate returns an error if one (and only one) option is not set.
func (c Config) Validate() error {
	numConfigured := 0
	for _, set := range []bool{c.Source != "", c.File != "", len(c.Files) > 0} {
		if set {
			numConfigured++
		}
	}

	switch {
	case numConfigured == 0:
		return errors.Errorf("javascript must be defined via 'file', " +
			"'files', or inline as 'source'")
	case numConfigured > 1:
		return errors.Errorf("javascript can be defined in only one of " +
			"'file', 'files', or inline as 'source'")
	}

	return nil
}

func defaultConfig() Config {
	return Config{
		TagOnException: "_js_exception",
	}
}
