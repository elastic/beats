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

package v2

import (
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

type simpleInputManager struct {
	configure func(*conf.C) (Input, error)
}

// ConfigureWith creates an InputManager that provides no extra logic and
// allows each input to fully control event collection and publishing in
// isolation. The function fn will be called for every input to be configured.
func ConfigureWith(fn func(*conf.C) (Input, error)) InputManager {
	return &simpleInputManager{configure: fn}
}

// Init is required to fulfil the input.InputManager interface.
// For the kafka input no special initialization is required.
func (*simpleInputManager) Init(grp unison.Group) error { return nil }

// Create builds a new Input instance from the given configuration, or returns
// an error if the configuration is invalid.
func (manager *simpleInputManager) Create(cfg *conf.C) (Input, error) {
	return manager.configure(cfg)
}
