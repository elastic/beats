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
	// Method to periodically fetch new events
	Fetch(m *MetricSet) ([]common.MapStr, error)
}

// Interface for each module
type Moduler interface {
	Setup() error
}
