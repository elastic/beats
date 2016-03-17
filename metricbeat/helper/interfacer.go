/**

= MetricSeter

== Timeout
Each metricseter must implement on his side to make sure the timeout is followed.
Otherwise it can lead to the issue that multiple calls for one metricset happen in parallel and pile up.

*/

package helper

import (
	"github.com/elastic/beats/libbeat/common"
)

/*

Implementation Background of Fetch

The initial idea was to use a Cancel function, but it was not implemented as it would add complexity by requiring
an identifier for each fetch call and it would have to be decide, which fetch call must be canceled.

*/

// Base configuration for each module/metricsets combination
type ModuleConfig struct {
	Hosts      []string `config:"hosts"`
	Period     string   `config:"period"`
	Timeout    string   `config:"timeout"`
	Module     string   `config:"module"`
	MetricSets []string `config:"metricsets"`
	Enabled    bool     `config:"enabled"`
	Selectors  []string `config:"selectors"`

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
	// Fetch is called on the predefined interval and does not take delays into account.
	Fetch(ms *MetricSet, host string) (common.MapStr, error)
}

// Interface for each module
type Moduler interface {
	// Raw ucfg config is passed. This allows each module to extract its own local config variables
	Setup(m *Module) error
}
