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

package xbuild

import (
	"fmt"

	"github.com/urso/magetools/clitool"
)

// Registry of available cross build environment providers.
type Registry struct {
	table map[OSArch]Provider
}

// Provider defines available functionality all cross build providers MUST implement.
type Provider interface {
	// Start prepare/starts up the provider
	Start() error

	// Stop shuts down the provider and cleans up resources if possible
	Stop() error

	// Executor creates the execution environment
	Executor(verbose bool) (clitool.Executor, error)

	// Shell starts an interactive shell if possible
	Shell() error
}

// OSArch tuple.
type OSArch struct {
	OS   string
	Arch string
}

// NewRegistry creates a new Regsitry.
func NewRegistry(tbl map[OSArch]Provider) *Registry {
	return &Registry{tbl}
}

// Find finds a provider by OS and Architecture name.
// Returns error if no provider can be found.
func (r *Registry) Find(os, arch string) (Provider, error) {
	p := r.table[OSArch{os, arch}]
	if p == nil {
		return nil, fmt.Errorf("No provider for %v:%v defined", os, arch)
	}
	return p, nil
}

func (r *Registry) With(os, arch string, fn func(Provider) error) error {
	p, err := r.Find(os, arch)
	if err != nil {
		return err
	}

	if err := p.Start(); err != nil {
		return err
	}
	defer p.Stop()

	return fn(p)
}
