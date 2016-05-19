package filter

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Filters struct {
	list []FilterRule
}

func New(config FilterPluginConfig) (*Filters, error) {

	filters := Filters{}

	for _, filter := range config {

		if len(filter) != 1 {
			return nil, fmt.Errorf("each filtering rule needs to have exactly one action, but found %d actions.", len(filter))
		}

		for filterName, cfg := range filter {

			constructor, exists := filterConstructors[filterName]
			if !exists {
				return nil, fmt.Errorf("the filtering rule %s doesn't exist", filterName)
			}

			plugin, err := constructor(cfg)
			if err != nil {
				return nil, err
			}

			filters.addRule(plugin)
		}
	}

	logp.Debug("filter", "filters: %v", filters)
	return &filters, nil
}

func (filters *Filters) addRule(filter FilterRule) {

	if filters.list == nil {
		filters.list = []FilterRule{}
	}
	filters.list = append(filters.list, filter)
}

// Applies a sequence of filtering rules and returns the filtered event
func (filters *Filters) Filter(event common.MapStr) common.MapStr {

	// Check if filters are set, just return event if not
	if len(filters.list) == 0 {
		return event
	}

	// clone the event at first, before starting filtering
	filtered := event.Clone()
	var err error

	for _, filter := range filters.list {
		filtered, err = filter.Filter(filtered)
		if err != nil {
			logp.Debug("filter", "fail to apply filtering rule %s: %s", filter, err)
		}
		if filtered == nil {
			// drop event
			return nil
		}
	}

	return filtered
}

func (filters Filters) String() string {
	s := []string{}

	for _, filter := range filters.list {

		s = append(s, filter.String())
	}
	return strings.Join(s, ", ")
}
