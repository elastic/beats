package filters

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

// The FilterPlugin interface needs to be implemented
// by all the filtering plugins.
type FilterPlugin interface {

	// New creates a new FilterPlugin instance with the
	// given name and configuration.
	New(name string, config map[string]interface{}) (FilterPlugin, error)

	// Filter executes the filter.
	Filter(event common.MapStr) (common.MapStr, error)

	// String returns the name of the filter.
	String() string

	// Type returns the type of the filter.
	Type() Filter
}

type Filter int

const (
	NopFilter Filter = iota
	SampleFilter
)

var FilterPluginNames = []string{
	"nop",
	"sample",
}

func (filter Filter) String() string {
	if int(filter) < 0 || int(filter) >= len(FilterPluginNames) {
		return "impossible"
	}
	return FilterPluginNames[filter]
}

func FilterFromName(name string) (Filter, error) {
	for i, pluginname := range FilterPluginNames {
		if name == pluginname {
			return Filter(i), nil
		}
	}
	return -1, fmt.Errorf("No filter named %s", name)
}

// Contains a list of the available filter plugins.
type FiltersList struct {
	filters map[Filter]FilterPlugin
}

var Filters FiltersList

func (filters FiltersList) Register(filter Filter, plugin FilterPlugin) {
	filters.filters[filter] = plugin
}

func (filters FiltersList) Get(filter Filter) FilterPlugin {
	return filters.filters[filter]
}

func init() {
	Filters = FiltersList{}
	Filters.filters = make(map[Filter]FilterPlugin)
}
