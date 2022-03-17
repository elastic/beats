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

package ecs

// Fields that describe the resources which container orchestrators manage or
// act upon.
type Orchestrator struct {
	// Name of the cluster.
	ClusterName string `ecs:"cluster.name"`

	// URL of the API used to manage the cluster.
	ClusterUrl string `ecs:"cluster.url"`

	// The version of the cluster.
	ClusterVersion string `ecs:"cluster.version"`

	// Orchestrator cluster type (e.g. kubernetes, nomad or cloudfoundry).
	Type string `ecs:"type"`

	// Organization affected by the event (for multi-tenant orchestrator
	// setups).
	Organization string `ecs:"organization"`

	// Namespace in which the action is taking place.
	Namespace string `ecs:"namespace"`

	// Name of the resource being acted upon.
	ResourceName string `ecs:"resource.name"`

	// Type of resource being acted upon.
	ResourceType string `ecs:"resource.type"`

	// API version being used to carry out the action
	ApiVersion string `ecs:"api_version"`
}
