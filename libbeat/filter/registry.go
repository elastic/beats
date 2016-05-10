package filter

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type FilterRule interface {
	Filter(event common.MapStr) (common.MapStr, error)
	String() string
}

type FilterConstructor func(config common.Config) (FilterRule, error)

var filterConstructors = map[string]FilterConstructor{}

func RegisterPlugin(name string, constructor FilterConstructor) error {

	logp.Debug("filter", "Register plugin %s", name)

	if _, exists := filterConstructors[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	filterConstructors[name] = constructor
	return nil
}
