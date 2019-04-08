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

package autodiscover

import (
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

// Register of autodiscover providers
type registry struct {
	// Lock to control concurrent read/writes
	lock sync.RWMutex
	// A map of provider name to ProviderBuilder.
	providers map[string]ProviderBuilder
	// A map of builder name to BuilderConstructor.
	builders map[string]BuilderConstructor
	// A map of appender name to AppenderBuilder.
	appenders map[string]AppenderBuilder

	logger *logp.Logger
}

// Registry holds all known autodiscover providers, they must be added to it to enable them for use
var Registry = NewRegistry()

// NewRegistry creates and returns a new Registry
func NewRegistry() *registry {
	return &registry{
		providers: make(map[string]ProviderBuilder, 0),
		builders:  make(map[string]BuilderConstructor, 0),
		appenders: make(map[string]AppenderBuilder, 0),
		logger:    logp.NewLogger("autodiscover"),
	}
}
