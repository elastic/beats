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

import (
	"fmt"
	"sync"
)

type PluginLoader func(p any) error

var (
	registry                   = map[string]PluginLoader{}
	ErrLoaderAlreadyRegistered = fmt.Errorf("already registered")
	m                          sync.Mutex
)

func Bundle(
	bundles ...map[string][]any,
) map[string][]any {
	ret := map[string][]any{}

	for _, bundle := range bundles {
		for name, plugins := range bundle {
			ret[name] = append(ret[name], plugins...)
		}
	}

	return ret
}

func MakePlugin(key string, ifc any) map[string][]any {
	return map[string][]any{
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
	m.Lock()
	defer m.Unlock()
	if l := registry[name]; l != nil {
		return fmt.Errorf("plugin loader '%v' %w", name, ErrLoaderAlreadyRegistered)
	}

	registry[name] = l
	return nil
}

func LoadPlugins(path string) error {
	// TODO: add flag to enable/disable plugins?
	return loadPlugins(path)
}

func GetLoader(name string) PluginLoader {
	m.Lock()
	defer m.Unlock()
	return registry[name]
}
