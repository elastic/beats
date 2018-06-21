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

package monitors

import (
	"errors"

	"github.com/elastic/beats/libbeat/plugin"
)

type monitorPlugin struct {
	name    string
	typ     Type
	builder ActiveBuilder
}

var pluginKey = "heartbeat.monitor"

func ActivePlugin(name string, b ActiveBuilder) map[string][]interface{} {
	return plugin.MakePlugin(pluginKey, monitorPlugin{name, ActiveMonitor, b})
}

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(monitorPlugin)
		if !ok {
			return errors.New("plugin does not match monitor plugin type")
		}

		return Registry.Register(p.name, p.typ, p.builder)
	})
}
