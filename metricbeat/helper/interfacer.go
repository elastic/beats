package helper

import (
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
	// Setup of MetricSeter
	// MetricSet which contains the MetricSeter is passed. This gives access to config
	// and the module.
	Setup(ms *MetricSet) error

	// Method to periodically fetch a new event from a host
	// Fetch is called for each host. In case where host does not exist, it can be transferred
	// differently in the setup to have a different meaning. An example here is for filesystem
	// of topbeat, where each host could be a filesystem.
	Fetch(ms *MetricSet, host string) (common.MapStr, error)
}

// Interface for each module
type Moduler interface {
	// Raw ucfg config is passed. This allows each module to extract its own local config variables
	Setup(m *Module) error
}
