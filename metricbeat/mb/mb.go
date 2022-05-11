// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package mb (short for Metricbeat) contains the public interfaces that are used
to implement Modules and their associated MetricSets.
*/
package mb

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/metricbeat/helper/dialer"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	// TimestampKey is the key used in events created by MetricSets to add their
	// own timestamp to an event. If a timestamp is not specified then the that
	// the fetch started will be used.
	TimestampKey string = "@timestamp"

	// ModuleDataKey is the key used in events created by MetricSets to add data
	// to an event that is common to the module. The data must be a
	// mapstr.M and when the final event is built the object will be stored
	// in the event under a key that is the module name.
	ModuleDataKey string = "_module"

	// NamespaceKey is used to define a different namespace for the metricset
	// This is useful for dynamic metricsets or metricsets which do not
	// put the name under the same name as the package. This is for example
	// the case in elasticsearch `node_stats` which puts the data under `node.stats`.
	NamespaceKey string = "_namespace"

	// RTTKey is used by a MetricSet to specify the round trip time (RTT), or
	// total amount of time, taken to collect the information in the event. The
	// data must be of type time.Duration otherwise the value is ignored.
	RTTKey string = "_rtt"
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
	rawConfig *conf.C
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

// WithConfig re-configures the module with the given raw configuration and returns a
// copy of the module.
// Intended to be called from module factories. Note that if metricsets are specified
// in the new configuration, those metricsets must already be registered with
// mb.Registry.
func (m *BaseModule) WithConfig(config conf.C) (*BaseModule, error) {
	var chkConfig struct {
		Module string `config:"module"`
	}
	if err := config.Unpack(&chkConfig); err != nil {
		return nil, errors.Wrap(err, "error parsing new module configuration")
	}

	// Don't allow module name change
	if chkConfig.Module != "" && chkConfig.Module != m.name {
		return nil, fmt.Errorf("cannot change module name from %v to %v", m.name, chkConfig.Module)
	}

	if err := config.SetString("module", -1, m.name); err != nil {
		return nil, errors.Wrap(err, "unable to set existing module name in new configuration")
	}

	newBM := &BaseModule{
		name:      m.name,
		rawConfig: &config,
	}

	if err := config.Unpack(&newBM.config); err != nil {
		return nil, errors.Wrap(err, "error parsing new module configuration")
	}

	return newBM, nil
}

// MetricSet interfaces

// MetricSet is the common interface for all MetricSet implementations. In
// addition to this interface, all MetricSets must implement a fetcher interface.
type MetricSet interface {
	ID() string     // Unique ID identifying a running MetricSet.
	Name() string   // Name returns the name of the MetricSet.
	Module() Module // Module returns the parent Module for the MetricSet.
	Host() string   // Host returns a hostname or other module specific value
	// that identifies a specific host or service instance from which to collect
	// metrics.
	HostData() HostData                  // HostData returns the parsed host data.
	Registration() MetricSetRegistration // Params used in registration.
	Metrics() *monitoring.Registry       // MetricSet specific metrics
	Logger() *logp.Logger                // MetricSet specific logger
}

// Closer is an optional interface that a MetricSet can implement in order to
// cleanup any resources it has open at shutdown.
type Closer interface {
	Close() error
}

// Reporter is used by a MetricSet to report events, errors, or errors with
// metadata. The methods return false if and only if publishing failed because
// the MetricSet is being closed.
//
// Deprecated: Use ReporterV2.
type Reporter interface {
	Event(event mapstr.M) bool               // Event reports a single successful event.
	ErrorWith(err error, meta mapstr.M) bool // ErrorWith reports a single error event with the additional metadata.
	Error(err error) bool                    // Error reports a single error event.
}

// ReportingMetricSet is a MetricSet that reports events or errors through the
// Reporter interface. Fetch is called periodically to collect events.
//
// Deprecated: Use ReportingMetricSetV2.
type ReportingMetricSet interface {
	MetricSet
	Fetch(r Reporter)
}

// PushReporter is used by a MetricSet to report events, errors, or errors with
// metadata. It provides a done channel used to signal that reporter should
// stop.
//
// Deprecated: Use PushReporterV2.
type PushReporter interface {
	Reporter

	// Done returns a channel that's closed when work done on behalf of this
	// reporter should be canceled.
	Done() <-chan struct{}
}

// PushMetricSet is a MetricSet that pushes events (rather than pulling them
// periodically via a Fetch callback). Run is invoked to start the event
// subscription and it should block until the MetricSet is ready to stop or
// the PushReporter's done channel is closed.
//
// Deprecated: Use PushMetricSetV2.
type PushMetricSet interface {
	MetricSet
	Run(r PushReporter)
}

// V2 Interfaces

// ReporterV2 is used by a MetricSet to report Events. The methods return false
// if and only if publishing failed because the MetricSet is being closed.
type ReporterV2 interface {
	Event(event Event) bool // Event reports a single successful event.
	Error(err error) bool
}

// PushReporterV2 is used by a MetricSet to report events, errors, or errors with
// metadata. It provides a done channel used to signal that reporter should
// stop.
type PushReporterV2 interface {
	ReporterV2

	// Done returns a channel that's closed when work done on behalf of this
	// reporter should be canceled.
	Done() <-chan struct{}
}

// ReportingMetricSetV2 is a MetricSet that reports events or errors through the
// ReporterV2 interface. Fetch is called periodically to collect events.
type ReportingMetricSetV2 interface {
	MetricSet
	Fetch(r ReporterV2)
}

// ReportingMetricSetV2Error is a MetricSet that reports events or errors through the
// ReporterV2 interface. Fetch is called periodically to collect events.
type ReportingMetricSetV2Error interface {
	MetricSet
	Fetch(r ReporterV2) error
}

// ReportingMetricSetV2WithContext is a MetricSet that reports events or errors through the
// ReporterV2 interface. Fetch is called periodically to collect events.
type ReportingMetricSetV2WithContext interface {
	MetricSet
	Fetch(ctx context.Context, r ReporterV2) error
}

// PushMetricSetV2 is a MetricSet that pushes events (rather than pulling them
// periodically via a Fetch callback). Run is invoked to start the event
// subscription and it should block until the MetricSet is ready to stop or
// the PushReporterV2's done channel is closed.
type PushMetricSetV2 interface {
	MetricSet
	Run(r PushReporterV2)
}

// PushMetricSetV2WithContext is a MetricSet that pushes events (rather than pulling them
// periodically via a Fetch callback). Run is invoked to start the event
// subscription and it should block until the MetricSet is ready to stop or
// the context is closed.
type PushMetricSetV2WithContext interface {
	MetricSet
	Run(ctx context.Context, r ReporterV2)
}

// HostData contains values parsed from the 'host' configuration. Other
// configuration data like protocols, usernames, and passwords may also be
// used to construct this HostData data. HostData also contains information when combined scheme are
// used, like doing HTTP request over a UNIX socket.
//
type HostData struct {
	Transport dialer.Builder // The transport builder to use when creating the connection.

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
	id           string
	name         string
	module       Module
	host         string
	hostData     HostData
	registration MetricSetRegistration
	metrics      *monitoring.Registry
	logger       *logp.Logger
}

func (b *BaseMetricSet) String() string {
	moduleName := "nil"
	if b.module != nil {
		moduleName = b.module.Name()
	}
	return fmt.Sprintf(`{name:"%v", module:"%v", hostData:%v, registration:%v}`,
		b.name, moduleName, b.hostData.String(), b.registration)
}

func (b *BaseMetricSet) GoString() string { return b.String() }

// ID returns the unique ID of the MetricSet.
func (b *BaseMetricSet) ID() string {
	return b.id
}

// Metrics returns the metrics registry.
func (b *BaseMetricSet) Metrics() *monitoring.Registry {
	return b.metrics
}

// Logger returns the logger.
func (b *BaseMetricSet) Logger() *logp.Logger {
	return b.logger
}

// Name returns the name of the MetricSet. It should not include the name of
// the module.
func (b *BaseMetricSet) Name() string {
	return b.name
}

// FullyQualifiedName returns the complete name of the MetricSet, including the
// name of the module.
func (b *BaseMetricSet) FullyQualifiedName() string {
	return b.Module().Name() + "/" + b.Name()
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

// Registration returns the parameters that were used when the MetricSet was
// registered with the registry.
func (b *BaseMetricSet) Registration() MetricSetRegistration {
	return b.registration
}

// Configuration types

// ModuleConfig is the base configuration data for all Modules.
//
// The Raw config option is used to enable raw fields in a metricset. This means
// the metricset fetches not only the predefined fields but add alls raw data under
// the raw namespace to the event.
type ModuleConfig struct {
	Hosts       []string      `config:"hosts"`
	Period      time.Duration `config:"period"     validate:"positive"`
	Timeout     time.Duration `config:"timeout"    validate:"positive"`
	Module      string        `config:"module"     validate:"required"`
	MetricSets  []string      `config:"metricsets"`
	Enabled     bool          `config:"enabled"`
	Raw         bool          `config:"raw"`
	Query       QueryParams   `config:"query"`
	ServiceName string        `config:"service.name"`
}

func (c ModuleConfig) String() string {
	return fmt.Sprintf(`{Module:"%v", MetricSets:%v, Enabled:%v, `+
		`Hosts:[%v hosts], Period:"%v", Timeout:"%v", Raw:%v, Query:%v}`,
		c.Module, c.MetricSets, c.Enabled, len(c.Hosts), c.Period, c.Timeout,
		c.Raw, c.Query)
}

func (c ModuleConfig) GoString() string { return c.String() }

// QueryParams is a convenient map[string]interface{} wrapper to implement the String interface which returns the
// values in common query params format (key=value&key2=value2) which is the way that the url package expects this
// params (without the initial '?')
type QueryParams map[string]interface{}

// String returns the values in common query params format (key=value&key2=value2) which is the way that the url
// package expects this params (without the initial '?')
func (q QueryParams) String() (s string) {
	u := url.Values{}

	for k, v := range q {
		if values, ok := v.([]interface{}); ok {
			for _, innerValue := range values {
				u.Add(k, fmt.Sprintf("%v", innerValue))
			}
		} else {
			//nil values in YAML shouldn't be stringified anyhow
			if v == nil {
				u.Add(k, "")
			} else {
				u.Add(k, fmt.Sprintf("%v", v))
			}
		}
	}

	return u.Encode()
}

// defaultModuleConfig contains the default values for ModuleConfig instances.
var defaultModuleConfig = ModuleConfig{
	Enabled: true,
	Period:  time.Second * 10,
}

// DefaultModuleConfig returns a ModuleConfig with the default values populated.
func DefaultModuleConfig() ModuleConfig {
	return defaultModuleConfig
}
