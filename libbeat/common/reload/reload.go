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

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// RegisterV2 is the special registry used for the V2 controller
var RegisterV2 = NewRegistry()

// InputRegName is the registation name for V2 inputs
const InputRegName = "input"

// OutputRegName is the registation name for V2 Outputs
const OutputRegName = "output"

// ConfigWithMeta holds a pair of config.C and optional metadata for it
type ConfigWithMeta struct {
	// Config to store
	Config *config.C

	// Meta data related to this config
	Meta *mapstr.Pointer

	// DiagCallback is a diagnostic handler associated with the underlying unit that maps to the config
	DiagCallback DiagnosticHandler
}

// ReloadableList provides a method to reload the configuration of a list of entities
type ReloadableList interface {
	Reload(configs []*ConfigWithMeta) error
}

// Reloadable provides a method to reload the configuration of an entity
type Reloadable interface {
	Reload(config *ConfigWithMeta) error
}

// ReloadableFunc wraps a custom function in order to implement the Reloadable interface.
type ReloadableFunc func(config *ConfigWithMeta) error

// DiagnosticHandler is an interface used to register diagnostic callbacks with the central management system
// This mostly exists to wrap the unit RegisterDiagnostic method
type DiagnosticHandler interface {
	Register(name string, description string, filename string, contentType string, callback func() []byte)
}

// Registry of reloadable objects and lists
type Registry struct {
	sync.RWMutex
	confsLists map[string]ReloadableList
	confs      map[string]Reloadable
}

// NewRegistry initializes and returns a reload registry
func NewRegistry() *Registry {
	return &Registry{
		confsLists: make(map[string]ReloadableList),
		confs:      make(map[string]Reloadable),
	}
}

// Register declares a reloadable object
func (r *Registry) Register(name string, obj Reloadable) error {
	r.Lock()
	defer r.Unlock()

	if obj == nil {
		return fmt.Errorf("got a nil object")
	}

	if r.nameTaken(name) {
		return fmt.Errorf("%s configuration list is already registered", name)
	}

	r.confs[name] = obj
	return nil
}

// RegisterList declares a reloadable list of configurations
func (r *Registry) RegisterList(name string, list ReloadableList) error {
	r.Lock()
	defer r.Unlock()

	if list == nil {
		return fmt.Errorf("got a nil object")
	}

	if r.nameTaken(name) {
		return fmt.Errorf("%s configuration is already registered", name)
	}

	r.confsLists[name] = list
	return nil
}

// MustRegister declares a reloadable object
func (r *Registry) MustRegister(name string, obj Reloadable) {
	if err := r.Register(name, obj); err != nil {
		panic(err)
	}
}

// MustRegisterOutput is a V2-specific registration function
// That declares a reloadable output
func (r *Registry) MustRegisterOutput(obj Reloadable) {
	if err := r.Register(OutputRegName, obj); err != nil {
		panic(err)
	}
}

// MustRegisterInput is a V2-specific registration function
// that declares a reloadable object list for a beat input
func (r *Registry) MustRegisterInput(list ReloadableList) {
	if err := r.RegisterList(InputRegName, list); err != nil {
		panic(err)
	}
}

// GetInputList is a V2-specific function
// That returns the reloadable list created for an input
func (r *Registry) GetInputList() ReloadableList {
	r.RLock()
	defer r.RUnlock()
	return r.confsLists[InputRegName]
}

// GetReloadableOutput is a V2-specific function
// That returns the reloader for the registered output
func (r *Registry) GetReloadableOutput() Reloadable {
	r.RLock()
	defer r.RUnlock()
	return r.confs[OutputRegName]
}

// GetRegisteredNames returns the list of names registered
func (r *Registry) GetRegisteredNames() []string {
	r.RLock()
	defer r.RUnlock()
	var names []string

	for name := range r.confs {
		names = append(names, name)
	}

	for name := range r.confsLists {
		names = append(names, name)
	}

	return names
}

// GetReloadable returns the reloadable object with the given name, nil if not found
func (r *Registry) GetReloadable(name string) Reloadable {
	r.RLock()
	defer r.RUnlock()
	return r.confs[name]
}

// GetReloadableList returns the reloadable list with the given name, nil if not found
func (r *Registry) GetReloadableList(name string) ReloadableList {
	r.RLock()
	defer r.RUnlock()
	return r.confsLists[name]
}

func (r *Registry) nameTaken(name string) bool {
	if _, ok := r.confs[name]; ok {
		return true
	}

	if _, ok := r.confsLists[name]; ok {
		return true
	}

	return false
}

// Reload calls the underlying function.
func (fn ReloadableFunc) Reload(config *ConfigWithMeta) error {
	return fn(config)
}
