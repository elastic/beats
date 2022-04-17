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

package builder

import (
	"errors"

	"github.com/menderesk/beats/v7/libbeat/autodiscover"
	p "github.com/menderesk/beats/v7/libbeat/plugin"
)

type builderPlugin struct {
	name    string
	builder autodiscover.BuilderConstructor
}

const pluginKey = "libbeat.autodiscover.builder"

// Plugin accepts a BuilderConstructor to be registered as a plugin
func Plugin(name string, b autodiscover.BuilderConstructor) map[string][]interface{} {
	return p.MakePlugin(pluginKey, builderPlugin{name, b})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		b, ok := ifc.(builderPlugin)
		if !ok {
			return errors.New("plugin does not match builder plugin type")
		}

		return autodiscover.Registry.AddBuilder(b.name, b.builder)
	})
}
