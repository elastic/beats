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
	"errors"
	"fmt"

	p "github.com/elastic/beats/v8/libbeat/plugin"
)

var (
	indexerKey = "libbeat.processor.kubernetes.indexer"
	matcherKey = "libbeat.processor.kubernetes.matcher"
)

type indexerPlugin struct {
	name        string
	constructor IndexerConstructor
}

func IndexerPlugin(name string, c IndexerConstructor) map[string][]interface{} {
	return p.MakePlugin(indexerKey, indexerPlugin{name, c})
}

type matcherPlugin struct {
	name        string
	constructor MatcherConstructor
}

func MatcherPlugin(name string, m MatcherConstructor) map[string][]interface{} {
	return p.MakePlugin(matcherKey, matcherPlugin{name, m})
}

func init() {
	p.MustRegisterLoader(indexerKey, func(ifc interface{}) error {
		i, ok := ifc.(indexerPlugin)
		if !ok {
			return errors.New("plugin does not match indexer plugin type")
		}

		name := i.name
		if Indexing.indexers[name] != nil {
			return fmt.Errorf("indexer type %v already registered", name)
		}

		Indexing.AddIndexer(name, i.constructor)
		return nil
	})

	p.MustRegisterLoader(matcherKey, func(ifc interface{}) error {
		m, ok := ifc.(matcherPlugin)
		if !ok {
			return errors.New("plugin does not match matcher plugin type")
		}

		name := m.name
		if Indexing.matchers[name] != nil {
			return fmt.Errorf("matcher type %v already registered", name)
		}

		Indexing.AddMatcher(name, m.constructor)
		return nil
	})
}
