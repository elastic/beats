// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"errors"
	"fmt"

	p "github.com/menderesk/beats/v7/libbeat/plugin"
)

var (
	indexerKey = "libbeat.processor.nomad.indexer"
	matcherKey = "libbeat.processor.nomad.matcher"
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
