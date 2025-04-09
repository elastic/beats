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
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// NewInputRegistry returns the *monitoring.Registry for metrics related to
// an input instance, identified by ID. If a registry with the given ID
// already exists, it is returned. Otherwise, a new registry is created.
//
// If a parent registry is provided, it will be used instead of the default
// 'dataset' monitoring namespace.
// If parent is nil, the default 'dataset' namespace is used. Therefore,
// inputType and id should be non-empty. If either is empty, the returned
// registry will not be registered in the global 'dataset' namespace. This will
// cause the metrics to not be available in the HTTP monitoring endpoint.
//
// The returned cancel function *must* be called when the input stops to
// unregister the metrics and prevent resource leaks.
//
// Deprecated. Use NewMetricsRegistry instead.
func NewInputRegistry(inputType, inputID string, optionalParent *monitoring.Registry) (reg *monitoring.Registry, cancel func()) {
	// Log the registration to ease tracking down duplicate ID registrations.
	// Logged at INFO rather than DEBUG since it is not in a hot path and having
	// the information available by default can short-circuit requests for debug
	// logs during support interactions.
	log := logp.NewLogger("metric_registry")

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
	registryID := sanitizeID(inputID)

	reg = parentRegistry.GetRegistry(registryID)
	if reg == nil {
		reg = parentRegistry.NewRegistry(registryID)
	} else {
		log.Warnw(fmt.Sprintf(
			"parent metrics registry already contains a %q registry, reusing it",
			registryID),
			"input_type", inputType,
			"input_id", inputID,
			"registry_id", registryID)
	}

	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputID)

	log.Infow("registering",
		"input_type", inputType,
		"input_id", inputID,
		"registry_id", registryID)

	return reg, func() {
		log.Infow("unregistering",
			"input_type", inputType,
			"input_id", inputID,
			"registry_id", registryID)
		parentRegistry.Remove(registryID)
	}
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(id, ".", "_")
}

func globalRegistry() *monitoring.Registry {
	return monitoring.GetNamespace("dataset").GetRegistry()
}

// MetricSnapshotJSON returns a snapshot of the input metric values from the
// global 'dataset' monitoring namespace and from the reg parameter
// encoded as a JSON array (pretty formatted). It's safe to pass in a nil
// reg.
func MetricSnapshotJSON(reg *monitoring.Registry) ([]byte, error) {
	return json.MarshalIndent(filteredSnapshot(globalRegistry(), reg, ""), "", "  ")
}

// NewMetricsRegistry creates a monitoring.Registry for an input.
//
// The metric registry is created on parent using inputID as the name,
// any '.' is replaced by '_'. The new registry is initialized with
// 'id: inputID' and 'input: inputType'.
//
// Call CancelMetricsRegistry to remove it from the parent registry and free up
// the associated resources.
func NewMetricsRegistry(
	inputID string,
	inputType string,
	parent *monitoring.Registry,
	log *logp.Logger) *monitoring.Registry {

	registryID := sanitizeID(inputID)
	reg := parent.GetRegistry(registryID)
	if reg == nil {
		reg = parent.NewRegistry(registryID)
	} else {
		log.Warnw(fmt.Sprintf(
			"parent metrics registry already contains a %q registry, reusing it",
			registryID),
			"registry_id", registryID,
			"input_type", inputType,
			"input_id", inputID)
	}

	// add the necessary information so the registry can be published by the
	// HTTP monitoring endpoint.
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputID)

	log.Named("metric_registry").Infow("registering",
		"registry_id", registryID,
		"input_id", inputID,
		"input_type", inputType)

	return reg
}

// CancelMetricsRegistry removes the metrics registry for inputID from parent.
func CancelMetricsRegistry(
	inputID string,
	inputType string,
	parent *monitoring.Registry,
	log *logp.Logger) {

	metricsID := sanitizeID(inputID)
	log.Named("metric_registry").Infow("unregistering",
		"registry_id", metricsID,
		"input_id", inputID,
		"input_type", inputType)

	parent.Remove(metricsID)
}
