package helper

import "github.com/elastic/beats/libbeat/common"

// Base configuration for each module/metricsets combination
type ModuleConfig struct {
	Hosts      []string
	Period     string
	Module     string
	MetricSets []string
	Enabled    bool
}

// Interface for each metric
type MetricSeter interface {
	// Setup needed for all upcoming fetches
	// Typically special config varialbes are loaded here
	Setup() error

	// Method to periodically fetch new events
	Fetch(m *MetricSet) ([]common.MapStr, error)

	// Cleanup when stopping metricset
	Cleanup() error
}

// Interface for each module
type Moduler interface {
	Setup() error
}
