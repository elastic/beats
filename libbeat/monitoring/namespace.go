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

import (
	"sync"
)

var namespaces = struct {
	sync.Mutex
	m map[string]*Namespace
}{
	m: make(map[string]*Namespace),
}

// Namespace contains the name of the namespace and it's registry
type Namespace struct {
	name     string
	registry *Registry
}

// GetNamespace gets the namespace with the given name.
// If the namespace does not exist yet, a new one is created.
func GetNamespace(name string) *Namespace {
	namespaces.Lock()
	defer namespaces.Unlock()

	n, ok := namespaces.m[name]
	if !ok {
		n = &Namespace{name: name}
		namespaces.m[name] = n
	}
	return n
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
