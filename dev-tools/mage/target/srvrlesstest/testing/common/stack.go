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

import "context"

// Stack is a created stack.
type Stack struct {
	// ID is the identifier of the instance.
	//
	// This must be the same ID used for requesting a stack.
	ID string `yaml:"id"`

	// Provisioner is the stack provisioner. See STACK_PROVISIONER environment
	// variable for the supported provisioners.
	Provisioner string `yaml:"provisioner"`

	// Version is the version of the stack.
	Version string `yaml:"version"`

	// Ready determines if the stack is ready to be used.
	Ready bool `yaml:"ready"`

	// Elasticsearch is the URL to communicate with elasticsearch.
	Elasticsearch string `yaml:"elasticsearch"`

	// Kibana is the URL to communication with kibana.
	Kibana string `yaml:"kibana"`

	// Username is the username.
	Username string `yaml:"username"`

	// Password is the password.
	Password string `yaml:"password"`

	// Internal holds internal information used by the provisioner.
	// Best to not touch the contents of this, and leave it be for
	// the provisioner.
	Internal map[string]interface{} `yaml:"internal"`
}

// Same returns true if other is the same stack as this one.
// Two stacks are considered the same if their provisioner and ID are the same.
func (s Stack) Same(other Stack) bool {
	return s.Provisioner == other.Provisioner &&
		s.ID == other.ID
}

// StackRequest request for a new stack.
type StackRequest struct {
	// ID is the unique ID for the stack.
	ID string `yaml:"id"`

	// Version is the version of the stack.
	Version string `yaml:"version"`
}

// StackProvisioner performs the provisioning of stacks.
type StackProvisioner interface {
	// Name returns the name of the stack provisioner.
	Name() string

	// SetLogger sets the logger for it to use.
	SetLogger(l Logger)

	// Create creates a stack.
	Create(ctx context.Context, request StackRequest) (Stack, error)

	// WaitForReady should block until the stack is ready or the context is cancelled.
	WaitForReady(ctx context.Context, stack Stack) (Stack, error)

	// Delete deletes the stack.
	Delete(ctx context.Context, stack Stack) error
}
