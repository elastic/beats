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
	"path/filepath"

	"github.com/elastic/go-ucfg"
)

type FileLoader func(name string, opts ...ucfg.Option) (*ucfg.Config, error)

func NewFlagFiles(
	cfg *ucfg.Config,
	extensions map[string]FileLoader,
	opts ...ucfg.Option,
) *FlagValue {
	return newFlagValue(cfg, opts, func(path string) (*ucfg.Config, error, error) {
		ext := filepath.Ext(path)
		loader := extensions[ext]
		if loader == nil {
			loader = extensions[""]
		}
		if loader == nil {
			// TODO: better error message?
			return nil, fmt.Errorf("no loader for file '%v' found", path), nil
		}
		cfg, err := loader(path, opts...)
		return cfg, err, nil
	})
}
