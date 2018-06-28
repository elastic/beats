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
	"github.com/elastic/beats/libbeat/autodiscover/appenders"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/providers"
	"github.com/elastic/beats/libbeat/feature"
)

// Registry is a wrapper over the new global registry, this will be removed in 7.0
var Registry = newRegistryWrapper{}

// newRegistryWrapper wraps the new global registry with the old style registry that were used by
// the autodiscover feature, this allow plugin that register at init() to still work.
type newRegistryWrapper struct{}

func (n *newRegistryWrapper) AddAppender(name string, factory appenders.Factory) {
	feature.MustRegister(appenders.Feature(name, factory, feature.Beta))
}

func (n *newRegistryWrapper) AddBuilder(name string, factory builder.Factory) {
	feature.MustRegister(builder.Feature(name, factory, feature.Beta))
}

func (n *newRegistryWrapper) AddProvider(name string, factory providers.Factory) {
	feature.MustRegister(providers.Feature(name, factory, feature.Beta))
}
