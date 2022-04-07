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

package feature

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/elastic/beats/v8/libbeat/logp"
)

type mapper map[string]map[string]Featurable

// Registry implements a global FeatureRegistry for any kind of feature in beats.
// feature are grouped by namespace, a namespace is a kind of plugin like outputs, inputs, or queue.
// The feature name must be unique.
type Registry struct {
	sync.RWMutex
	namespaces mapper
	log        *logp.Logger
}

// NewRegistry returns a new registry.
func NewRegistry() *Registry {
	return &Registry{
		namespaces: make(mapper),
		log:        logp.NewLogger("registry"),
	}
}

// Register registers a new feature into a specific namespace, namespace are lazy created.
// Feature name must be unique.
func (r *Registry) Register(feature Featurable) error {
	r.Lock()
	defer r.Unlock()

	ns := normalize(feature.Namespace())
	name := normalize(feature.Name())

	if feature.Factory() == nil {
		return fmt.Errorf("feature '%s' cannot be registered with a nil factory", name)
	}

	// Lazy create namespaces
	_, found := r.namespaces[ns]
	if !found {
		r.namespaces[ns] = make(map[string]Featurable)
	}

	f, found := r.namespaces[ns][name]
	if found {
		if featuresEqual(feature, f) {
			// Allow both old style and new style of plugin to work together.
			r.log.Debugw(
				"ignoring, feature '%s' is already registered in the namespace '%s'",
				name,
				ns,
			)
			return nil
		}

		return fmt.Errorf(
			"could not register new feature '%s' in namespace '%s', feature name must be unique",
			name,
			ns,
		)
	}

	r.log.Debugw(
		"registering new feature",
		"namespace",
		ns,
		"name",
		name,
	)

	r.namespaces[ns][name] = feature

	return nil
}

// Unregister removes a feature from the registry.
func (r *Registry) Unregister(namespace, name string) error {
	r.Lock()
	defer r.Unlock()
	ns := normalize(namespace)

	v, found := r.namespaces[ns]
	if !found {
		return fmt.Errorf("unknown namespace named '%s'", ns)
	}

	_, found = v[name]
	if !found {
		return fmt.Errorf("unknown feature '%s' in namespace '%s'", name, ns)
	}

	delete(r.namespaces[ns], name)
	return nil
}

// Lookup searches for a Feature by the namespace-name pair.
func (r *Registry) Lookup(namespace, name string) (Featurable, error) {
	r.RLock()
	defer r.RUnlock()

	ns := normalize(namespace)
	n := normalize(name)

	v, found := r.namespaces[ns]
	if !found {
		return nil, fmt.Errorf("unknown namespace named '%s'", ns)
	}

	m, found := v[n]
	if !found {
		return nil, fmt.Errorf("unknown feature '%s' in namespace '%s'", n, ns)
	}

	return m, nil
}

// LookupAll returns all the features for a specific namespace.
func (r *Registry) LookupAll(namespace string) ([]Featurable, error) {
	r.RLock()
	defer r.RUnlock()

	ns := normalize(namespace)

	v, found := r.namespaces[ns]
	if !found {
		return nil, fmt.Errorf("unknown namespace named '%s'", ns)
	}

	list := make([]Featurable, len(v))
	c := 0
	for _, feature := range v {
		list[c] = feature
		c++
	}

	return list, nil
}

// Overwrite allow to replace an existing feature with a new implementation.
func (r *Registry) Overwrite(feature Featurable) error {
	_, err := r.Lookup(feature.Namespace(), feature.Name())
	if err == nil {
		err := r.Unregister(feature.Namespace(), feature.Name())
		if err != nil {
			return err
		}
	}

	return r.Register(feature)
}

// Size returns the number of registered features in the registry.
func (r *Registry) Size() int {
	r.RLock()
	defer r.RUnlock()

	c := 0
	for _, namespace := range r.namespaces {
		c += len(namespace)
	}

	return c
}

func featuresEqual(f1, f2 Featurable) bool {
	// There is no safe way to compare function in go,
	// but since the function pointers are global it should be stable.
	if f1.Name() == f2.Name() &&
		f1.Namespace() == f2.Namespace() &&
		reflect.ValueOf(f1.Factory()).Pointer() == reflect.ValueOf(f2.Factory()).Pointer() {
		return true
	}

	return false
}

func normalize(s string) string {
	return strings.ToLower(s)
}
