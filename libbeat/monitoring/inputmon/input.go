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
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

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
// Deprecated. Use input/v2.NewMetricsRegistry instead.
func NewInputRegistry(inputType, inputID string, optionalParent *monitoring.Registry) (reg *monitoring.Registry, cancel func()) {
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
	metricsID := sanitizeID(inputID)

	reg = parentRegistry.GetRegistry(metricsID)
	if reg == nil {
		reg = parentRegistry.NewRegistry(metricsID)
	}

	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputID)

	// Log the registration to ease tracking down duplicate ID registrations.
	// Logged at INFO rather than DEBUG since it is not in a hot path and having
	// the information available by default can short-circuit requests for debug
	// logs during support interactions.
	log := logp.NewLogger("metric_registry")

	log.Infow("registering",
		"input_type", inputType,
		"input_id", inputID,
		"metrics_id", metricsID)

	// TODO: test adding and removing an new input to ensure the registry is removed
	return reg, func() {
		log.Infow("unregistering", "input_type", inputType,
			"input_id", inputID,
			"metrics_id", metricsID)
		parentRegistry.Remove(metricsID)
	}
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(id, ".", "_")
}

func globalRegistry() *monitoring.Registry {
	return monitoring.GetNamespace("dataset").GetRegistry()
}

// MetricSnapshotJSON returns a snapshot of the input metric values from the
// global 'dataset' monitoring namespace and from the localReg parameter
// encoded as a JSON array (pretty formatted). It's safe to pass in a nil
// localReg.
func MetricSnapshotJSON(reg *monitoring.Registry) ([]byte, error) {
	return json.MarshalIndent(filteredSnapshot(globalRegistry(), reg, ""), "", "  ")
}

// NewMetricsRegistry creates a monitoring.Registry for an input.
//
// The metric registry is created on parent with
// name 'inputID' ('.' are replaced by '_') and populated with 'id: inputID' and
// 'input: inputType'.
//
// Call CancelMetricsRegistry to remove it from the parent registry and free up
// the associated resources.
func NewMetricsRegistry(
	inputID string,
	inputType string,
	parent *monitoring.Registry,
	log *logp.Logger) *monitoring.Registry {

	metricsID := sanitizeID(inputID)
	reg := parent.GetRegistry(metricsID)
	if reg == nil {
		reg = parent.NewRegistry(metricsID)
	}

	// add the necessary information so the registry can be published by the
	// HTTP monitoring endpoint.
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputID)

	log.Named("metric_registry").Infow("registering",
		"metrics_id", metricsID,
		"input_id", inputID,
		"input_type", inputType)

	return reg
}

func CancelMetricsRegistry(
	inputID string,
	inputType string,
	reg *monitoring.Registry,
	log *logp.Logger) {

	metricsID := sanitizeID(inputID)
	log.Named("metric_registry").Infow("unregistering",
		"metrics_id", metricsID,
		"input_id", inputID,
		"input_type", inputType)

	reg.Remove(metricsID)
}
