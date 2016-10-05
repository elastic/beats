/*
Package mb (short for Metricbeat) contains the public interfaces that are used
to implement Modules and their associated MetricSets.
*/
package mb

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	ModuleData string = "_module"
)

// Module interfaces

// Module is the common interface for all Module implementations.
type Module interface {
	Name() string                      // Name returns the name of the Module.
	Config() ModuleConfig              // Config returns the ModuleConfig used to create the Module.
	UnpackConfig(to interface{}) error // UnpackConfig unpacks the raw module config to the given object.
}

// BaseModule implements the Module interface.
//
// When a Module needs to store additional data or provide methods to its
// MetricSets, it can embed this type into another struct to satisfy the
// Module interface requirements.
type BaseModule struct {
	name      string
	config    ModuleConfig
	rawConfig *common.Config
}

// Name returns the name of the Module.
func (m *BaseModule) Name() string { return m.name }

// Config returns the ModuleConfig used to create the Module.
func (m *BaseModule) Config() ModuleConfig { return m.config }

// UnpackConfig unpacks the raw module config to the given object.
func (m *BaseModule) UnpackConfig(to interface{}) error {
	return m.rawConfig.Unpack(to)
}

// MetricSet interfaces

// MetricSet is the common interface for all MetricSet implementations. In
// addition to this interface, all MetricSets must implement either
// EventFetcher or EventsFetcher (but not both).
type MetricSet interface {
	Name() string   // Name returns the name of the MetricSet.
	Module() Module // Module returns the parent Module for the MetricSet.
	Host() string   // Host returns a hostname or other module specific value
	// that identifies a specific host or service instance from which to collect
	// metrics.
}

// EventFetcher is a MetricSet that returns a single event when collecting data.
type EventFetcher interface {
	MetricSet
	Fetch() (common.MapStr, error)
}

// EventsFetcher is a MetricSet that returns a multiple events when collecting
// data.
type EventsFetcher interface {
	MetricSet
	Fetch() ([]common.MapStr, error)
}

// BaseMetricSet implements the MetricSet interface.
//
// The BaseMetricSet type can be embedded into another struct to satisfy the
// MetricSet interface requirements, leaving only the Fetch() method to be
// implemented to have a complete MetricSet implementation.
type BaseMetricSet struct {
	name   string
	module Module
	host   string
}

// Name returns the name of the MetricSet. It should not include the name of
// the module.
func (b *BaseMetricSet) Name() string {
	return b.name
}

// Module returns the parent Module for the MetricSet.
func (b *BaseMetricSet) Module() Module {
	return b.module
}

// Host returns the hostname or other module specific value that identifies a
// specific host or service instance from which to collect metrics.
func (b *BaseMetricSet) Host() string {
	return b.host
}

// Configuration types

// ModuleConfig is the base configuration data for all Modules.
type ModuleConfig struct {
	Hosts      []string                `config:"hosts"`
	Period     time.Duration           `config:"period"     validate:"positive"`
	Timeout    time.Duration           `config:"timeout"    validate:"positive"`
	Module     string                  `config:"module"     validate:"required"`
	MetricSets []string                `config:"metricsets" validate:"required"`
	Enabled    bool                    `config:"enabled"`
	Filters    processors.PluginConfig `config:"filters"`

	common.EventMetadata `config:",inline"` // Fields and tags to add to events.
}

// defaultModuleConfig contains the default values for ModuleConfig instances.
var defaultModuleConfig = ModuleConfig{
	Enabled: true,
	Period:  time.Second * 10,
	Timeout: time.Second,
}

// DefaultModuleConfig returns a ModuleConfig with the default values populated.
func DefaultModuleConfig() ModuleConfig {
	return defaultModuleConfig
}
