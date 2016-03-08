package helper

import (
	"github.com/urso/ucfg"

	"github.com/elastic/beats/libbeat/common"
)

// Base configuration for each module/metricsets combination
type ModuleConfig struct {
	Hosts      []string `config:"hosts"`
	Period     string   `config:"period"`
	Module     string   `config:"module"`
	MetricSets []string `config:"metricsets"`
	Enabled    bool     `config:"enabled"`

	common.EventMetadata `config:",inline"` // Fields and tags to add to events.
}

// Interface for each metric
type MetricSeter interface {
	// Method to periodically fetch new events
	Fetch(m *MetricSet) ([]common.MapStr, error)
}

// Interface for each module
type Moduler interface {
	// Raw ucfg config is passed. This allows each module to extract its own local config variables
	Setup(cfg *ucfg.Config) error
}
