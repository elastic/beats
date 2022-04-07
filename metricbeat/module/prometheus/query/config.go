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

package query

import (
	"errors"

	"github.com/elastic/beats/v8/libbeat/common"
)

// Config defines the "query" metricset's configuration
type Config struct {
	Queries      []QueryConfig `config:"queries"`
	DefaultQuery QueryConfig   `config:"default_path"`
}

// QueryConfig is used to make an API request.
type QueryConfig struct {
	Path   string        `config:"path"`
	Params common.MapStr `config:"params"`
	Name   string        `config:"name"`
}

func defaultConfig() Config {
	return Config{
		DefaultQuery: QueryConfig{
			Path: "/api/v1/query",
			Name: "default",
		},
	}
}

// Validate for Prometheus "query" metricset config
func (p *QueryConfig) Validate() error {
	if p.Name == "" {
		return errors.New("`name` can not be empty in path configuration")
	}

	if p.Path == "" {
		return errors.New("`path` can not be empty in path configuration")
	}

	return nil
}
