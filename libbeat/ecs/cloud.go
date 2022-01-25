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

// Fields related to the cloud or infrastructure the events are coming from.
type Cloud struct {
	// Name of the cloud provider. Example values are aws, azure, gcp, or
	// digitalocean.
	Provider string `ecs:"provider"`

	// Availability zone in which this host, resource, or service is located.
	AvailabilityZone string `ecs:"availability_zone"`

	// Region in which this host, resource, or service is located.
	Region string `ecs:"region"`

	// Instance ID of the host machine.
	InstanceID string `ecs:"instance.id"`

	// Instance name of the host machine.
	InstanceName string `ecs:"instance.name"`

	// Machine type of the host machine.
	MachineType string `ecs:"machine.type"`

	// The cloud account or organization id used to identify different entities
	// in a multi-tenant environment.
	// Examples: AWS account id, Google Cloud ORG Id, or other unique
	// identifier.
	AccountID string `ecs:"account.id"`

	// The cloud account name or alias used to identify different entities in a
	// multi-tenant environment.
	// Examples: AWS account name, Google Cloud ORG display name.
	AccountName string `ecs:"account.name"`

	// The cloud service name is intended to distinguish services running on
	// different platforms within a provider, eg AWS EC2 vs Lambda, GCP GCE vs
	// App Engine, Azure VM vs App Server.
	// Examples: app engine, app service, cloud run, fargate, lambda.
	ServiceName string `ecs:"service.name"`

	// The cloud project identifier.
	// Examples: Google Cloud Project id, Azure Project id.
	ProjectID string `ecs:"project.id"`

	// The cloud project name.
	// Examples: Google Cloud Project name, Azure Project name.
	ProjectName string `ecs:"project.name"`
}
