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

package fbprovider

import (
	"context"

	"go.opentelemetry.io/collector/confmap"

	"github.com/elastic/beats/v7/libbeat/otelbeat/providers"
)

const schemeName = "fb"

type fbProvider struct{}

// NewFactory returns a provider factory that loads filebeat configuration
func NewFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(newProvider)
}

func newProvider(confmap.ProviderSettings) confmap.Provider {
	return &fbProvider{}
}

// Retrieve retrieves the beat configuration file and constructs otel config
// uri here is the filepath of the beat config
func (*fbProvider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return providers.LoadConfig(uri, schemeName)
}

func (*fbProvider) Scheme() string {
	return schemeName
}

func (*fbProvider) Shutdown(context.Context) error {
	return nil
}
