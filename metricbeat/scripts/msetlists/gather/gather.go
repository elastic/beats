package gather

import (
	"strings"

	"github.com/elastic/beats/metricbeat/mb"
)

// DefaultMetricsets returns a JSON array of all registered default metricsets
// It depends upon the calling library to actually import or register the metricsets.
func DefaultMetricsets() map[string][]string {
	// List all registered modules and metricsets.
	var defaultMap = make(map[string][]string)
	for _, mod := range mb.Registry.Modules() {
		metricSets, err := mb.Registry.DefaultMetricSets(mod)
		if err != nil && !strings.Contains(err.Error(), "no default metricset for") {
			continue
		}
		defaultMap[mod] = metricSets
	}

	return defaultMap

}
