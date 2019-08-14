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

package appenders

import (
	"errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	p "github.com/elastic/beats/libbeat/plugin"
)

type appenderPlugin struct {
	name     string
	appender autodiscover.AppenderBuilder
}

var pluginKey = "libbeat.autodiscover.appender"

// Plugin accepts a AppenderBuilder to be registered as a plugin
func Plugin(name string, appender autodiscover.AppenderBuilder) map[string][]interface{} {
	return p.MakePlugin(pluginKey, appenderPlugin{name, appender})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		app, ok := ifc.(appenderPlugin)
		if !ok {
			return errors.New("plugin does not match appender plugin type")
		}

		return autodiscover.Registry.AddAppender(app.name, app.appender)
	})
}
