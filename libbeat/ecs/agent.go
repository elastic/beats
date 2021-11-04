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

// The agent fields contain the data about the software entity, if any, that
// collects, detects, or observes events on a host, or takes measurements on a
// host.
// Examples include Beats. Agents may also run on observers. ECS agent.* fields
// shall be populated with details of the agent running on the host or observer
// where the event happened or the measurement was taken.
type Agent struct {
	// Version of the agent.
	Version string `ecs:"version"`

	// Extended build information for the agent.
	// This field is intended to contain any build information that a data
	// source may provide, no specific formatting is required.
	BuildOriginal string `ecs:"build.original"`

	// Custom name of the agent.
	// This is a name that can be given to an agent. This can be helpful if for
	// example two Filebeat instances are running on the same host but a human
	// readable separation is needed on which Filebeat instance data is coming
	// from.
	// If no name is given, the name is often left empty.
	Name string `ecs:"name"`

	// Type of the agent.
	// The agent type always stays the same and should be given by the agent
	// used. In case of Filebeat the agent would always be Filebeat also if two
	// Filebeat instances are run on the same machine.
	Type string `ecs:"type"`

	// Unique identifier of this agent (if one exists).
	// Example: For Beats this would be beat.id.
	ID string `ecs:"id"`

	// Ephemeral identifier of this agent (if one exists).
	// This id normally changes across restarts, but `agent.id` does not.
	EphemeralID string `ecs:"ephemeral_id"`
}
