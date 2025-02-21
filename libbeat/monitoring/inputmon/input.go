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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// NewInputRegistry returns a new *monitoring.Registry for metrics related to
// an input instance. The registry named ID.
// The returned cancel function *must* be called when the input stops to
// unregister the metrics and prevent resource leaks.
//
// If a parent registry is provided, it will be used instead of the default
// 'dataset' monitoring namespace.
//
// If parent is nil, inputType and id must be non-empty otherwise the metrics
// will not be published.
//
// This function will panic if a registry named id was already registered.
func NewInputRegistry(inputType, id string, parent *monitoring.Registry) (reg *monitoring.Registry, cancel func()) {
	// Use the default registry unless one was provided (this would be for testing).
	parentRegistry := parent
	if parentRegistry == nil {
		parentRegistry = globalRegistry()
	}

	// If an ID has not been assigned to an input then metrics cannot be exposed
	// in the global metric registry. The returned registry still behaves the same.
	if (id == "" || inputType == "") && parentRegistry == globalRegistry() {
		// Null route metrics without ID or input type.
		parentRegistry = monitoring.NewRegistry()
	}

	// Sanitize dots from the id because they created nested objects within
	// the monitoring registry, and we want a consistent flat level of nesting
	key := sanitizeID(id)

	// Log the registration to ease tracking down duplicate ID registrations.
	// Logged at INFO rather than DEBUG since it is not in a hot path and having
	// the information available by default can short-circuit requests for debug
	// logs during support interactions.
	log := logp.NewLogger("metric_registry")
	// Make an orthogonal ID to allow tracking register/deregister pairs.
	uuid := uuid.Must(uuid.NewV4()).String()
	log.Infow("registering", "input_type", inputType, "id", id, "key", key, "uuid", uuid)

	reg = parentRegistry.NewRegistry(key)
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(id)

	return reg, func() {
		log.Infow("unregistering", "input_type", inputType, "id", id, "key", key, "uuid", uuid)
		parentRegistry.Remove(key)
	}
}

func sanitizeID(id string) string {
	return strings.ReplaceAll(id, ".", "_")
}

func globalRegistry() *monitoring.Registry {
	return monitoring.GetNamespace("dataset").GetRegistry()
}

// MetricSnapshotJSON returns a snapshot of the input metric values from the
// global 'dataset' and from the beat monitoring namespace from the beatInfo
// instance. It returns a pretty formated JSON array as a byte slice.
func MetricSnapshotJSON(beatInfo beat.Info) ([]byte, error) {
	return json.MarshalIndent(
		filteredSnapshot(
			globalRegistry(),
			beatInfo.Monitoring.Namespace.GetRegistry(),
			""),
		"",
		"  ")
}
