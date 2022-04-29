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

package processors

import (
	"errors"

	p "github.com/elastic/beats/v7/libbeat/plugin"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type processorPlugin struct {
	name   string
	constr Constructor
}

var pluginKey = "libbeat.processor"

func Plugin(name string, c Constructor) map[string][]interface{} {
	return p.MakePlugin(pluginKey, processorPlugin{name, c})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(processorPlugin)
		if !ok {
			return errors.New("plugin does not match processor plugin type")
		}

		return registry.Register(p.name, p.constr)
	})
}

type Constructor func(config *config.C) (Processor, error)

var registry = NewNamespace()

func RegisterPlugin(name string, constructor Constructor) {
	logp.L().Named(logName).Debugf("Register plugin %s", name)

	err := registry.Register(name, constructor)
	if err != nil {
		panic(err)
	}
}
