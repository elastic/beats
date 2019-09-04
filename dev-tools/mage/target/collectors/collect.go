package collectors

import (
	metricbeat "github.com/elastic/beats/metricbeat/scripts/mage"
)

//CollectDocs creates the documentation under docs/
func CollectDocs() error {
	return metricbeat.CollectDocs()
}
