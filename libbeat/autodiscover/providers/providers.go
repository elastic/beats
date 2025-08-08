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

package providers

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers/docker"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers/jolokia"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers/kubernetes"
)

var KnownProviders = map[string]autodiscover.ProviderBuilder{
	docker.ProviderName:     docker.AutodiscoverBuilder,
	jolokia.ProviderName:    jolokia.AutodiscoverBuilder,
	kubernetes.ProviderName: kubernetes.AutodiscoverBuilder,
}

func AddKnownProviders(r *autodiscover.Registry, providers map[string]autodiscover.ProviderBuilder) error {
	for name, builder := range providers {
		if err := r.AddProvider(name, builder); err != nil {
			return fmt.Errorf("error adding provider %s: %w", name, err)
		}
	}
	return nil
}
