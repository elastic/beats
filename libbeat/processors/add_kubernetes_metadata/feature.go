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

import "github.com/elastic/beats/libbeat/feature"

var (
	indexerNamespace = "libbeat.processor.add_kubernetes_metadata.indexer"
	matcherNamespace = "libbeat.processor.add_kubernetes_metadata.matcher"
)

// IndexerPlugin is a backward compatible method to create a new indexer.
func IndexerPlugin(name string, c IndexerConstructor) *feature.Feature {
	return IndexerFeature(name, c, feature.NewDetails(name, "", feature.Undefined))
}

// MatcherPlugin is a backward compatible method to create a new matcher.
func MatcherPlugin(name string, m MatcherConstructor) *feature.Feature {
	return MatcherFeature(name, m, feature.NewDetails(name, "", feature.Undefined))
}

// IndexerFeature creates a new Indexer for kubernetes metadata.
func IndexerFeature(
	name string,
	factory IndexerConstructor,
	description feature.Describer,
) *feature.Feature {
	return feature.New(indexerNamespace, name, factory, description)
}

// MatcherFeature creates a new Matcher for kubernetes metadata.
func MatcherFeature(
	name string,
	factory MatcherConstructor,
	description feature.Describer,
) *feature.Feature {
	return feature.New(matcherNamespace, name, factory, description)
}
