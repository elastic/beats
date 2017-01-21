/*
Package mb (short for Metricbeat) contains the public interfaces that are used
to implement Modules and their associated MetricSets.
*/
package mb

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	// ModuleData is the key used in events created by MetricSets to add data
	// to an event that is common to the module. The data must be a
	// common.MapStr and when the final event is built the object will be stored
	// in the event under a key that is the module name.
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

func (m *BaseModule) String() string {
	return fmt.Sprintf(`{name:"%v", config:%v}`, m.name, m.config.String())
}

func (m *BaseModule) GoString() string { return m.String() }

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
	HostData() HostData // HostData returns the parsed host data.
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

// HostData contains values parsed from the 'host' configuration. Other
// configuration data like protocols, usernames, and passwords may also be
// used to construct this HostData data.
type HostData struct {
	URI          string // The full URI that should be used in connections.
	SanitizedURI string // A sanitized version of the URI without credentials.

	// Parts of the URI.

	Host     string // The host and possibly port.
	User     string // Username
	Password string // Password
}

func (h HostData) String() string {
	return fmt.Sprintf(`{SanitizedURI:"%v", Host:"%v"}`, h.SanitizedURI, h.Host)
}

func (h HostData) GoString() string { return h.String() }

// BaseMetricSet implements the MetricSet interface.
//
// The BaseMetricSet type can be embedded into another struct to satisfy the
// MetricSet interface requirements, leaving only the Fetch() method to be
// implemented to have a complete MetricSet implementation.
type BaseMetricSet struct {
	name     string
	module   Module
	host     string
	hostData HostData
}

func (b *BaseMetricSet) String() string {
	moduleName := "nil"
	if b.module != nil {
		moduleName = b.module.Name()
	}
	return fmt.Sprintf(`{name:"%v", module:"%v", hostData:%v}`,
		b.name, moduleName, b.hostData.String())
}

func (b *BaseMetricSet) GoString() string { return b.String() }

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

// HostData returns the parsed host data.
func (b *BaseMetricSet) HostData() HostData {
	return b.hostData
}

// Configuration types

// ModuleConfig is the base configuration data for all Modules.
//
// The Raw config option is used to enable raw fields in a metricset. This means
// the metricset fetches not only the predefined fields but add alls raw data under
// the raw namespace to the event.
type ModuleConfig struct {
	Hosts      []string                `config:"hosts"`
	Period     time.Duration           `config:"period"     validate:"positive"`
	Timeout    time.Duration           `config:"timeout"    validate:"positive"`
	Module     string                  `config:"module"     validate:"required"`
	MetricSets []string                `config:"metricsets" validate:"required"`
	Enabled    bool                    `config:"enabled"`
	Filters    processors.PluginConfig `config:"filters"`
	Raw        bool                    `config:"raw"`

	common.EventMetadata `config:",inline"` // Fields and tags to add to events.
}

func (c ModuleConfig) String() string {
	return fmt.Sprintf(`{Module:"%v", MetricSets:%v, Enabled:%v, `+
		`Hosts:[%v hosts], Period:"%v", Timeout:"%v", Raw:%v, Fields:%v, `+
		`FieldsUnderRoot:%v, Tags:%v}`,
		c.Module, c.MetricSets, c.Enabled, len(c.Hosts), c.Period, c.Timeout,
		c.Raw, c.Fields, c.FieldsUnderRoot, c.Tags)
}

func (c ModuleConfig) GoString() string { return c.String() }

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
