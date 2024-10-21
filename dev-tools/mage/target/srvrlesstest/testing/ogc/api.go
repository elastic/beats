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

package ogc

import "github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"

// Layout definition for `ogc layout import`.
type Layout struct {
	Name          string            `yaml:"name"`
	Provider      string            `yaml:"provider"`
	InstanceSize  string            `yaml:"instance_size"`
	RunsOn        string            `yaml:"runs_on"`
	RemotePath    string            `yaml:"remote_path"`
	Scale         int               `yaml:"scale"`
	Username      string            `yaml:"username"`
	SSHPrivateKey string            `yaml:"ssh_private_key"`
	SSHPublicKey  string            `yaml:"ssh_public_key"`
	Ports         []string          `yaml:"ports"`
	Tags          []string          `yaml:"tags"`
	Labels        map[string]string `yaml:"labels"`
	Scripts       string            `yaml:"scripts"`
}

// Machine definition returned by `ogc up`.
type Machine struct {
	ID            int    `yaml:"id"`
	InstanceID    string `yaml:"instance_id"`
	InstanceName  string `yaml:"instance_name"`
	InstanceState string `yaml:"instance_state"`
	PrivateIP     string `yaml:"private_ip"`
	PublicIP      string `yaml:"public_ip"`
	Layout        Layout `yaml:"layout"`
	Create        string `yaml:"created"`
}

// LayoutOS defines the minimal information for a mapping of an OS to the
// provider, instance size, and runs on for that OS.
type LayoutOS struct {
	OS           define.OS
	Provider     string
	InstanceSize string
	RunsOn       string
	Username     string
	RemotePath   string
}
