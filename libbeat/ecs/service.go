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

// The service fields describe the service for or from which the data was
// collected.
// These fields help you find and correlate logs for a specific service and
// version.
type Service struct {
	// Identifies the environment where the service is running.
	// If the same service runs in different environments (production, staging,
	// QA, development, etc.), the environment can identify other instances of
	// the same service. Can also group services and applications from the same
	// environment.
	Environment string `ecs:"environment"`

	// Unique identifier of the running service. If the service is comprised of
	// many nodes, the `service.id` should be the same for all nodes.
	// This id should uniquely identify the service. This makes it possible to
	// correlate logs and metrics for one specific service, no matter which
	// particular node emitted the event.
	// Note that if you need to see the events from one specific host of the
	// service, you should filter on that `host.name` or `host.id` instead.
	ID string `ecs:"id"`

	// Name of the service data is collected from.
	// The name of the service is normally user given. This allows for
	// distributed services that run on multiple hosts to correlate the related
	// instances based on the name.
	// In the case of Elasticsearch the `service.name` could contain the
	// cluster name. For Beats the `service.name` is by default a copy of the
	// `service.type` field if no name is specified.
	Name string `ecs:"name"`

	// Name of a service node.
	// This allows for two nodes of the same service running on the same host
	// to be differentiated. Therefore, `service.node.name` should typically be
	// unique across nodes of a given service.
	// In the case of Elasticsearch, the `service.node.name` could contain the
	// unique node name within the Elasticsearch cluster. In cases where the
	// service doesn't have the concept of a node name, the host name or
	// container name can be used to distinguish running instances that make up
	// this service. If those do not provide uniqueness (e.g. multiple
	// instances of the service running on the same host) - the node name can
	// be manually set.
	NodeName string `ecs:"node.name"`

	// The type of the service data is collected from.
	// The type can be used to group and correlate logs and metrics from one
	// service type.
	// Example: If logs or metrics are collected from Elasticsearch,
	// `service.type` would be `elasticsearch`.
	Type string `ecs:"type"`

	// Current state of the service.
	State string `ecs:"state"`

	// Version of the service the data was collected from.
	// This allows to look at a data set only for a specific version of a
	// service.
	Version string `ecs:"version"`

	// Ephemeral identifier of this service (if one exists).
	// This id normally changes across restarts, but `service.id` does not.
	EphemeralID string `ecs:"ephemeral_id"`

	// Address where data about this service was collected from.
	// This should be a URI, network address (ipv4:port or [ipv6]:port) or a
	// resource path (sockets).
	Address string `ecs:"address"`
}
