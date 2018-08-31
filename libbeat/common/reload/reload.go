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

package reload

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/common"
)

var register = newRegistry()

// ConfigWithMeta holds a pair of common.Config and optional metadata for it
type ConfigWithMeta struct {
	// Config to store
	Config *common.Config

	// Meta data related to this config
	Meta *common.MapStrPointer
}

// ReloadableList provides a method to reload the configuration of a list of entities
type ReloadableList interface {
	Reload(configs []*ConfigWithMeta) error
}

// Reloadable provides a method to reload the configuration of an entity
type Reloadable interface {
	Reload(config *ConfigWithMeta) error
}

type registry struct {
	sync.RWMutex
	confsLists map[string]ReloadableList
	confs      map[string]Reloadable
}

func newRegistry() registry {
	return registry{
		confsLists: make(map[string]ReloadableList),
		confs:      make(map[string]Reloadable),
	}
}

// MustRegister declares a reloadable object
func MustRegister(name string, r Reloadable) {
	register.Lock()
	defer register.Unlock()

	if r == nil {
		panic("Got a nil object")
	}

	if _, ok := register.confs[name]; ok {
		panic(fmt.Sprintf("%s configuration is already registered", name))
	}

	register.confs[name] = r
}

// MustRegisterList declares a reloadable list of configurations
func MustRegisterList(name string, list ReloadableList) {
	register.Lock()
	defer register.Unlock()

	if list == nil {
		panic("Got a nil object")
	}

	if _, ok := register.confsLists[name]; ok {
		panic(fmt.Sprintf("%s configuration list is already registered", name))
	}

	register.confsLists[name] = list
}

// Get returns the reloadable object with the given name, nil if not found
func Get(name string) Reloadable {
	register.RLock()
	defer register.RUnlock()
	return register.confs[name]
}

// GetList returns the reloadable list with the given name, nil if not found
func GetList(name string) ReloadableList {
	register.RLock()
	defer register.RUnlock()
	return register.confsLists[name]
}
