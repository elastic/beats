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

package plugin

import "fmt"

type PluginLoader func(p interface{}) error

var registry = map[string]PluginLoader{}

func Bundle(
	bundles ...map[string][]interface{},
) map[string][]interface{} {
	ret := map[string][]interface{}{}

	for _, bundle := range bundles {
		for name, plugins := range bundle {
			ret[name] = append(ret[name], plugins...)
		}
	}

	return ret
}

func MakePlugin(key string, ifc interface{}) map[string][]interface{} {
	return map[string][]interface{}{
		key: {ifc},
	}
}

func MustRegisterLoader(name string, l PluginLoader) {
	err := RegisterLoader(name, l)
	if err != nil {
		panic(err)
	}
}

func RegisterLoader(name string, l PluginLoader) error {
	if l := registry[name]; l != nil {
		return fmt.Errorf("plugin loader '%v' already registered", name)
	}

	registry[name] = l
	return nil
}

func LoadPlugins(path string) error {
	// TODO: add flag to enable/disable plugins?
	return loadPlugins(path)
}
