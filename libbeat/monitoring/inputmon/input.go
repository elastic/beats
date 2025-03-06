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

package inputmon

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type inputRegistry struct {
	mu         sync.RWMutex
	registries map[string]*monitoring.Registry
}

var registeredInputs = inputRegistry{
	registries: make(map[string]*monitoring.Registry),
}

// RegisterMetrics adds reg to the collection of registries to be returned by
// the `/inputs/` endpoint. The registry must have at least a `id` and `input`
// string variables, otherwise the registry is rejected and an error is
// returned.
// If an id/inputType registry has benn already registered, it'll be overridden.
// When the input finishes, it should call UnregisterMetrics to
// release the associated resources.
func RegisterMetrics(id string, reg *monitoring.Registry) error {
	idValid := validStringVar(reg.Get("id"))
	inputValid := validStringVar(reg.Get("input"))

	errMgs := ""
	if !idValid {
		errMgs = "'id' empty or absent"
	}
	if !inputValid {
		errMgs += ", 'input' empty or absent"
	}
	if errMgs != "" {
		return errors.New("invalid metrics registry: " + errMgs)
	}

	registeredInputs.Set(id, reg)

	return nil
}

// UnregisterMetrics removes the registry identified by id/inputType.
func UnregisterMetrics(id string) {
	registeredInputs.Del(id)
}

func (i *inputRegistry) Get(id string) (*monitoring.Registry, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()

	v, found := i.registries[id]
	return v, found
}

func (i *inputRegistry) Set(id string, reg *monitoring.Registry) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.registries[id] = reg
}

func (i *inputRegistry) Del(id string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.registries, id)
}

func (i *inputRegistry) CollectStructSnapshot() map[string]map[string]any {
	registeredInputRegistries := map[string]map[string]any{}

	registeredInputs.mu.Lock()
	for id, reg := range registeredInputs.registries {
		registeredInputRegistries[id] = monitoring.CollectStructSnapshot(
			reg, monitoring.Full, false)
	}
	registeredInputs.mu.Unlock()

	return registeredInputRegistries
}

func validStringVar(v monitoring.Var) bool {
	if v != nil {
		if s, ok := v.(*monitoring.String); ok {
			return s.Get() != ""
		}
	}

	return false
}

// NewInputRegistry returns the *monitoring.Registry for metrics related to
// an input instance, identified by ID. If a registry with the given ID
// already exists, it is returned. Otherwise, a new registry is created.
// If a parent registry is provided, it will be used instead of the default
// 'dataset' monitoring namespace.
// If parent is nil, inputType and id must be non-empty. Otherwise, the metrics
// will not be published.
//
// The returned cancel function *must* be called when the input stops to
// unregister the metrics and prevent resource leaks.
//
// This function might panic.
// Deprecated.
func NewInputRegistry(inputType, inputID string, optionalParent *monitoring.Registry) (reg *monitoring.Registry, cancel func()) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("inoutmon.NewInputRegistry panic: %+v", r))
		}
	}()

	// Use the default registry unless one was provided (this would be for testing).
	parentRegistry := optionalParent
	if parentRegistry == nil {
		parentRegistry = globalRegistry()
	}

	// If an ID has not been assigned to an input then metrics cannot be exposed
	// in the global metric registry. The returned registry still behaves the same.
	if (inputID == "" || inputType == "") && parentRegistry == globalRegistry() {
		// Null route metrics without ID or input type.
		parentRegistry = monitoring.NewRegistry()
	}

	// Sanitize dots from the id because they created nested objects within
	// the monitoring registry, and we want a consistent flat level of nesting
	registryName := sanitizeID(inputID)

	reg = parentRegistry.GetRegistry(registryName)
	if reg == nil {
		reg = parentRegistry.NewRegistry(registryName)
	}

	uid := EnhanceInputRegistry(reg, inputID, inputType)

	// Log the registration to ease tracking down duplicate ID registrations.
	// Logged at INFO rather than DEBUG since it is not in a hot path and having
	// the information available by default can short-circuit requests for debug
	// logs during support interactions.
	log := logp.NewLogger("metric_registry")
	log.Infow("registering", "input_type", inputType, "id", inputID, "key", registryName, "uuid", uid)

	return reg, func() {
		log.Infow("unregistering", "input_type", inputType, "id", inputID, "key", registryName, "uuid", uid)
		parentRegistry.Remove(registryName)
	}
}

func EnhanceInputRegistry(reg *monitoring.Registry, inputID string, inputType string) string {
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputID)
	// Make an orthogonal ID to allow tracking register/deregister pairs.
	uid := uuid.Must(uuid.NewV4()).String()

	return uid
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(id, ".", "_")
}

func globalRegistry() *monitoring.Registry {
	return monitoring.GetNamespace("dataset").GetRegistry()
}

// MetricSnapshotJSON returns a snapshot of the input metric values from the
// global 'dataset' monitoring namespace encoded as a JSON array (pretty formatted).
func MetricSnapshotJSON() ([]byte, error) {
	return json.MarshalIndent(filteredSnapshot(globalRegistry(), ""), "", "  ")
}
