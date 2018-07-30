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

package flag

import (
	"fmt"
	"strings"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/internal/parse"
)

// NewFlagKeyValue implements the flag.Value interface for
// capturing ucfg.Config settings from command line arguments.
// Configuration options follow the argument name and must be in the form of
// "key=value". Using 'D' as command line flag for example, options on command line
// must be given as:
//
// -D key1=value -D key=value
//
// Note: the space between command line option and key is required by the flag
// package to parse command line flags correctly.
//
// Note: it's valid to use a key multiple times. If keys are used multiple
// times, values get overwritten. The last known value for some key will be stored
// in the generated configuration.
//
// The type of value must be any of bool, uint, int, float, or string. Any kind
// of array or object syntax is not supported.
//
// If autoBool is enabled (default if Config or ConfigVar is used), keys without
// value are converted to bool variable with value being true.
func NewFlagKeyValue(cfg *ucfg.Config, autoBool bool, opts ...ucfg.Option) *FlagValue {
	return newFlagValue(cfg, opts, func(arg string) (*ucfg.Config, error, error) {
		var key string
		var val interface{}
		var err error

		args := strings.SplitN(arg, "=", 2)
		if len(args) < 2 {
			if !autoBool || len(args) == 0 {
				err := fmt.Errorf("argument '%v' is empty ", arg)
				return nil, err, err
			}

			key = arg
			val = true
		} else {
			key = args[0]
			if args[1] == "" {
				return nil, nil, nil
			}

			val, err = parse.Value(args[1])
			if err != nil {
				return nil, err, err
			}
		}

		tmp := map[string]interface{}{key: val}
		cfg, err := ucfg.NewFrom(tmp, opts...)
		return cfg, err, err
	})
}
