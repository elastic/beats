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

package codec

import (
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

type Factory func(beat.Info, *common.Config) (Codec, error)

type Config struct {
	Namespace common.ConfigNamespace `config:",inline"`
}

var codecs = map[string]Factory{}

func RegisterType(name string, gen Factory) {
	if _, exists := codecs[name]; exists {
		panic(fmt.Sprintf("output codec '%v' already registered ", name))
	}
	codecs[name] = gen
}

func CreateEncoder(info beat.Info, cfg Config) (Codec, error) {
	// default to json codec
	codec := "json"
	if name := cfg.Namespace.Name(); name != "" {
		codec = name
	}

	factory := codecs[codec]
	if factory == nil {
		return nil, fmt.Errorf("'%v' output codec is not available", codec)
	}
	return factory(info, cfg.Namespace.Config())
}
