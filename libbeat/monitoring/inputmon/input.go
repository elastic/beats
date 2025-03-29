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

	"github.com/gofrs/uuid/v5"

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
// Deprecated. Use beat.Info.Monitoring.InputHTTPMetrics.RegisterMetrics instead.
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
	registryName := sanitizeID(inputID)

	reg = parentRegistry.GetRegistry(registryName)
	if reg == nil {
		reg = parentRegistry.NewRegistry(registryName)
	}

	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputID)

	// Log the registration to ease tracking down duplicate ID registrations.
	// Logged at INFO rather than DEBUG since it is not in a hot path and having
	// the information available by default can short-circuit requests for debug
	// logs during support interactions.
	log := logp.NewLogger("metric_registry")

	// Make an orthogonal ID to allow tracking register/deregister pairs.
	var uid string
	if rawID, err := uuid.NewV4(); err != nil {
		log.Errorf("failed to register metrics for '%s', id: %s,: %v",
			inputType, inputID, err)
	} else {
		uid = rawID.String()
	}
	log.Infow("registering", "input_type", inputType, "id", inputID, "key", registryName, "uuid", uid)

	return reg, func() {
		log.Infow("unregistering", "input_type", inputType, "id", inputID, "key", registryName, "uuid", uid)
		parentRegistry.Remove(registryName)
	}
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(id, ".", "_")
}

func globalRegistry() *monitoring.Registry {
	return monitoring.GetNamespace("dataset").GetRegistry()
}

// MetricSnapshotJSON returns a snapshot of the input metric values from the
// global 'dataset' monitoring namespace and from the inputMetrics parameter
// encoded as a JSON array (pretty formatted). It's safe to pass in a nil
// inputMetrics.
func MetricSnapshotJSON(inputMetrics StructSnapshotCollector) ([]byte, error) {
	snapCollector := inputMetrics
	if snapCollector == nil {
		snapCollector = &noopStructSnapshotCollector{}
	}

	return json.MarshalIndent(filteredSnapshot(globalRegistry(), snapCollector, ""), "", "  ")
}
