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

package monitoring

import "sync"

var namespaces = NewNamespaces()

// Namespace contains the name of the namespace and it's registry
type Namespace struct {
	name     string
	registry *Registry
}

func newNamespace(name string) *Namespace {
	n := &Namespace{
		name: name,
	}
	return n
}

// GetNamespace gets the namespace with the given name.
// If the namespace does not exist yet, a new one is created.
func GetNamespace(name string) *Namespace {
	return namespaces.Get(name)
}

// SetRegistry sets the registry of the namespace
func (n *Namespace) SetRegistry(r *Registry) {
	n.registry = r
}

// GetRegistry gets the registry of the namespace
func (n *Namespace) GetRegistry() *Registry {
	if n.registry == nil {
		n.registry = NewRegistry()
	}
	return n.registry
}

// Namespaces is a list of Namespace structs
type Namespaces struct {
	sync.Mutex
	namespaces map[string]*Namespace
}

// NewNamespaces creates a new namespaces list
func NewNamespaces() *Namespaces {
	return &Namespaces{
		namespaces: map[string]*Namespace{},
	}
}

// Get returns the namespace for the given key. If the key does not exist, new namespace is created.
func (n *Namespaces) Get(key string) *Namespace {
	n.Lock()
	defer n.Unlock()
	if namespace, ok := n.namespaces[key]; ok {
		return namespace
	}

	n.namespaces[key] = newNamespace(key)
	return n.namespaces[key]
}
