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
	"fmt"
	"time"
)

// Config stores the options of an input.
type Config struct {
	// Paths stores the paths to the journal files to be read.
	Paths []string `config:"paths"`
	// MaxBackoff is the limit of the backoff time.
	Backoff time.Duration `config:"backoff" validate:"min=0,nonzero"`
	// Backoff is the current interval to wait before
	// attemting to read again from the journal.
	BackoffFactor int `config:"backoff_factor" validate:"min=1"`
	// BackoffFactor is the multiplier of Backoff.
	MaxBackoff time.Duration `config:"max_backoff" validate:"min=0,nonzero"`
	// Seek is the method to read from journals.
	Seek string `config:"seek"`
}

var (
	// DefaultConfig is the defaults for an inputs
	DefaultConfig = Config{
		Backoff:       1 * time.Second,
		BackoffFactor: 2,
		MaxBackoff:    60 * time.Second,
		Seek:          "tail",
	}
)

// Validate check the configuration of the input.
func (c *Config) Validate() error {
	correctSeek := false
	for _, s := range []string{"cursor", "head", "tail"} {
		if c.Seek == s {
			correctSeek = true
		}
	}

	if !correctSeek {
		return fmt.Errorf("incorrect value for seek: %s. possible values: cursor, head, tail", c.Seek)
	}

	return nil
}
