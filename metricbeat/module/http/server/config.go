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

package server

import (
	"errors"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type HttpServerConfig struct {
	Paths       []PathConfig `config:"server.paths"`
	DefaultPath PathConfig   `config:"server.default_path"`
}

type PathConfig struct {
	Path      string   `config:"path"`
	Fields    mapstr.M `config:"fields"`
	Namespace string   `config:"namespace"`
}

func defaultHttpServerConfig() HttpServerConfig {
	return HttpServerConfig{
		DefaultPath: PathConfig{
			Path:      "/",
			Namespace: "server",
		},
	}
}

func (p PathConfig) Validate() error {
	if p.Namespace == "" {
		return errors.New("`namespace` can not be empty in path configuration")
	}

	if p.Path == "" {
		return errors.New("`path` can not be empty in path configuration")
	}

	return nil
}
