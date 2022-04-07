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

package input

import (
	"fmt"

	"github.com/elastic/beats/v8/filebeat/channel"
	"github.com/elastic/beats/v8/filebeat/input/file"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

type Context struct {
	States   []file.State
	Done     chan struct{}
	BeatDone chan struct{}
	Meta     map[string]string
}

// Factory is used to register functions creating new Input instances.
type Factory = func(config *common.Config, connector channel.Connector, context Context) (Input, error)

var registry = make(map[string]Factory)

func Register(name string, factory Factory) error {
	logp.Info("Registering input factory")
	if name == "" {
		return fmt.Errorf("Error registering input: name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("Error registering input '%v': factory cannot be empty", name)
	}
	if _, exists := registry[name]; exists {
		return fmt.Errorf("Error registering input '%v': already registered", name)
	}

	registry[name] = factory
	logp.Info("Successfully registered input")

	return nil
}

func GetFactory(name string) (Factory, error) {
	if _, exists := registry[name]; !exists {
		return nil, fmt.Errorf("Error creating input. No such input type exist: '%v'", name)
	}
	return registry[name], nil
}
