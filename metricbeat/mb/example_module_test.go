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

package mb_test

import (
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "example" module.
	if err := mb.Registry.AddModule("example", NewModule); err != nil {
		panic(err)
	}
}

type Module struct {
	mb.BaseModule
	Protocol string
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Unpack additional configuration options.
	config := struct {
		Protocol string `config:"protocol"`
	}{
		Protocol: "udp",
	}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &Module{BaseModule: base, Protocol: config.Protocol}, nil
}

// ExampleModuleFactory demonstrates how to register a custom ModuleFactory
// and unpack additional configuration data.
func ExampleModuleFactory() {}
