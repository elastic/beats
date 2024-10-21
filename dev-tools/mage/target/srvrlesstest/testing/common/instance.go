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

package common

import (
	"context"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
)

type ProvisionerType uint32

const (
	ProvisionerTypeVM ProvisionerType = iota
	ProvisionerTypeK8SCluster
)

// Instance represents a provisioned instance.
type Instance struct {
	// Provider is the instance provider for the instance.
	// See INSTANCE_PROVISIONER environment variable for the supported providers.
	Provider string `yaml:"provider"`
	// ID is the identifier of the instance.
	//
	// This must be the same ID of the OSBatch.
	ID string `yaml:"id"`
	// Name is the nice-name of the instance.
	Name string `yaml:"name"`
	// Provisioner is the instance provider for the instance.
	// See INSTANCE_PROVISIONER environment variable for the supported Provisioner.
	Provisioner string `yaml:"provisioner"`
	// IP is the IP address of the instance.
	IP string `yaml:"ip"`
	// Username is the username used to SSH to the instance.
	Username string `yaml:"username"`
	// RemotePath is the based path used for performing work on the instance.
	RemotePath string `yaml:"remote_path"`
	// Internal holds internal information used by the provisioner.
	// Best to not touch the contents of this, and leave it be for
	// the provisioner.
	Internal map[string]interface{} `yaml:"internal"`
}

// InstanceProvisioner performs the provisioning of instances.
type InstanceProvisioner interface {
	// Name returns the name of the instance provisioner.
	Name() string

	// Type returns the type of the provisioner.
	Type() ProvisionerType

	// SetLogger sets the logger for it to use.
	SetLogger(l Logger)

	// Supported returns true of false if the provisioner supports the given batch.
	Supported(batch define.OS) bool

	// Provision brings up the machines.
	//
	// The provision should re-use already prepared instances when possible.
	Provision(ctx context.Context, cfg Config, batches []OSBatch) ([]Instance, error)

	// Clean cleans up all provisioned resources.
	Clean(ctx context.Context, cfg Config, instances []Instance) error
}
