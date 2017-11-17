package add_kubernetes_metadata

import (
	"errors"
	"fmt"

	p "github.com/elastic/beats/libbeat/plugin"
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
