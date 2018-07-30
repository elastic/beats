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

package add_kubernetes_metadata

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
)

var (
	indexerNamespace = "libbeat.processor.add_kubernetes_metadata.indexer"
	matcherNamespace = "libbeat.processor.add_kubernetes_metadata.matcher"
)

type indexerPlugin struct {
	Default bool
	Config  *common.Config
	Factory IndexerConstructor
}

type indexerFunc func() *indexerPlugin

type matcherPlugin struct {
	Default bool
	Config  *common.Config
	Factory MatcherConstructor
}

type matcherFunc func() *matcherPlugin

// IndexerFeature creates a new Indexer for kubernetes metadata.
func IndexerFeature(
	name string,
	factory IndexerConstructor,
	d bool,
	config *common.Config,
	description feature.Describer,
) *feature.Feature {
	return feature.New(indexerNamespace, name, func() *indexerPlugin {
		return &indexerPlugin{
			Default: d,
			Config:  config,
			Factory: factory,
		}
	}, description)
}

// MatcherFeature creates a new Matcher for kubernetes metadata.
func MatcherFeature(
	name string,
	factory MatcherConstructor,
	d bool,
	config *common.Config,
	description feature.Describer,
) *feature.Feature {
	return feature.New(matcherNamespace, name, func() *matcherPlugin {
		return &matcherPlugin{
			Default: d,
			Config:  config,
			Factory: factory,
		}
	}, description)
}

// FindIndexerFactory returns the factory for a specific indexer.
func FindIndexerFactory(name string) (IndexerConstructor, error) {
	f, err := feature.Registry.Lookup(indexerNamespace, name)
	if err != nil {
		return nil, err
	}

	pluginFactory, ok := f.Factory().(indexerFunc)

	if !ok {
		return nil, fmt.Errorf("invalid indexer type, received: %T", f.Factory())
	}

	plugin := pluginFactory()
	return plugin.Factory, nil
}

// FindDefaultIndexersConfigs returns the list of the default indexers and their config.
func FindDefaultIndexersConfigs() (configs []map[string]common.Config, err error) {
	features, err := feature.Registry.LookupAll(indexerNamespace)
	if err != nil {
		return nil, err
	}

	for _, f := range features {
		pluginFactory, ok := f.Factory().(indexerFunc)

		if !ok {
			return nil, fmt.Errorf("invalid indexer type, received: %T", f.Factory())
		}

		plugin := pluginFactory()

		if plugin.Default {
			configs = append(configs, map[string]common.Config{f.Name(): *plugin.Config})
		}
	}

	return configs, nil
}

// FindMatcherFactory returns the factory for a specific indexer.
func FindMatcherFactory(name string) (MatcherConstructor, error) {
	f, err := feature.Registry.Lookup(matcherNamespace, name)
	if err != nil {
		return nil, err
	}

	pluginFactory, ok := f.Factory().(matcherFunc)

	if !ok {
		return nil, fmt.Errorf("invalid matcher type, received: %T", f.Factory())
	}

	plugin := pluginFactory()
	return plugin.Factory, nil
}

// FindDefaultMatchersConfigs return the default matchers and their default config.
func FindDefaultMatchersConfigs() (configs []map[string]common.Config, err error) {
	features, err := feature.Registry.LookupAll(indexerNamespace)
	if err != nil {
		return nil, err
	}

	for _, f := range features {
		pluginFactory, ok := f.Factory().(matcherFunc)

		if !ok {
			return nil, fmt.Errorf("invalid matcher type, received: %T", f.Factory())
		}

		plugin := pluginFactory()

		if plugin.Default {
			configs = append(configs, map[string]common.Config{f.Name(): *plugin.Config})
		}
	}

	return configs, nil
}
