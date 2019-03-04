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

// FileLoader is used by NewFlagFiles to define customer file loading functions
// for different file extensions.
type FileLoader func(name string, opts ...ucfg.Option) (*ucfg.Config, error)

// NewFlagFiles create a new flag, that will load external configurations file
// when being used. Configurations loaded from multiple files will be merged
// into one common Config object.  If cfg is not nil, then the loaded
// configurations will be merged into cfg.
// The extensions parameter define custom file loaders for different file
// extensions. If extensions contains an entry with key "", then this loader
// will be used as default fallback.
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
