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

package v2

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/go-concert/unison"
)

// InputManager creates and maintains actions and background processes for an
// input type.
// The InputManager is used to create inputs. The InputManager can provide
// additional functionality like coordination between input of the same type,
// custom functionality for querying or caching shared information, application
// of common settings not unique to a particular input type, or require a more
// specific Input interface to be implemented by the actual input.
type InputManager interface {
	// Init signals to InputManager to initialize internal resources.
	// The mode tells the input manager if the Beat is actually running the inputs or
	// if inputs are only configured for testing/validation purposes.
	Init(grp unison.Group) error

	// Create builds a new Input instance from the given configuation, or returns
	// an error if the configuation is invalid.
	// The input must establish any connection for data collection yet. The Beat
	// will use the Test/Run methods of the input.
	Create(*conf.C) (Input, error)
}

// Input is a configured input object that can be used to test or start
// the actual data collection.
type Input interface {
	// Name reports the input name.
	//
	// XXX: check if/how we can remove this method. Currently it is required for
	// compatibility reasons with existing interfaces in libbeat, autodiscovery
	// and filebeat.
	Name() string

	// Test checks the configuration and runs additional checks if the Input can
	// actually collect data for the given configuration (e.g. check if host/port or files are
	// accessible).
	Test(TestContext) error

	// Run starts the data collection. Run must return an error only if the
	// error is fatal making it impossible for the input to recover.
	Run(Context, beat.PipelineConnector) error
}

// Context provides the Input Run function with common environmental
// information and services.
type Context struct {
	// Logger provides a structured logger to inputs. The logger is initialized
	// with labels that will identify logs for the input.
	Logger *logp.Logger

	// The input ID.
	ID string

	// The input ID without name. Some inputs append sourcename, we need the id to be untouched
	// https://github.com/elastic/beats/blob/43d80af2aea60b0c45711475d114e118d90c4581/filebeat/input/v2/input-cursor/input.go#L118
	IDWithoutName string

	// Name is the input name, sometimes referred as input type.
	Name string

	// Agent provides additional Beat info like instance ID or beat name.
	Agent beat.Info

	// Cancelation is used by Beats to signal the input to shut down.
	Cancelation Canceler

	// StatusReporter provides a method to update the status of the underlying unit
	// that maps to the config. Note: Under standalone execution of Filebeat this is
	// expected to be nil.
	StatusReporter status.StatusReporter

	// monitoringRegistry is the registry collecting metrics for the input using
	// this context.
	monitoringRegistry *monitoring.Registry
	// monitoringRegistryCancel removes the registry from its parent and from
	// the HTTP monitoring endpoint.
	monitoringRegistryCancel func()
}

// NewContext creates a new context.
func NewContext(
	id,
	idWithoutName,
	inputType string,
	agent beat.Info,
	cancelation Canceler,
	statusReporter status.StatusReporter,
	reg *monitoring.Registry,
	unreg func(),
	log *logp.Logger) Context {

	return Context{
		ID:            id,
		IDWithoutName: idWithoutName,
		Name:          inputType,

		Agent:          agent,
		Cancelation:    cancelation,
		StatusReporter: statusReporter,

		Logger: log,

		monitoringRegistry:       reg,
		monitoringRegistryCancel: unreg,
	}
}

// NewMetricsRegistry creates and registers a monitoring.Registry for an input
// with the HTTP monitoring endpoint. It returns the metrics registry and a
// function to unregister it.
//
// The metric registry is created on the metrics namespace from beatInfo with
// name 'inputId' and populated with 'id: inputId' and 'input: inputType'.
// An error is logged if the new registry cannot be registered with the HTTP
// monitoring endpoint.
//
// The unregister function removes the registry the beatInfo monitoring
// namespace as well as from the monitoring HTTP endpoint.
func NewMetricsRegistry(
	inputId string,
	inputType string,
	beatInfo *beat.Info,
	log *logp.Logger) (*monitoring.Registry, func()) {

	parentRegistry := beatInfo.Monitoring.NamespaceRegistry()
	metricsID := strings.ReplaceAll(inputId, ".", "_")
	reg := parentRegistry.GetRegistry(metricsID)
	if reg == nil {
		reg = parentRegistry.NewRegistry(metricsID)
	}

	// add the necessary information so the registry can be published by the
	// HTTP monitoring endpoint.
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(inputId)

	// register to be published by the HTTP monitoring endpoint.
	err := beatInfo.Monitoring.InputHTTPMetrics.RegisterMetrics(reg)
	if err != nil {
		log.Errorf("failed to register metrics for '%s', id: %s,: %v",
			inputType, inputId, err)
	}

	metricsLog := logp.NewLogger("metric_registry")
	var uid string
	if uidv4, err := uuid.NewV4(); err != nil {
		metricsLog.Errorf(
			"failed to generate uuid to track input metrics register/unregister: %v",
			err)
	} else {
		uid = uidv4.String()
	}

	// Log the registration to ease tracking down duplicate ID registrations.
	// Logged at INFO rather than DEBUG since it is not in a hot path and having
	// the information available by default can short-circuit requests for debug
	// logs during support interactions.
	metricsLog.Infow("registering",
		"input_type", inputType,
		"id", inputId,
		"registry_name", metricsID,
		"uuid", uid)

	unreg := func() {
		metricsLog.Infow("unregistering",
			"input_type", inputType,
			"id", inputId,
			"registry_name",
			metricsID, "uuid", uid)
		parentRegistry.Remove(metricsID)
		// it's safe to make this call even if registering them failed.
		beatInfo.Monitoring.InputHTTPMetrics.UnregisterMetrics(metricsID)
	}

	return reg, unreg
}

func (c *Context) UpdateStatus(status status.Status, msg string) {
	if c.StatusReporter != nil {
		c.Logger.Debugf("updating status, status: '%s', message: '%s'", status.String(), msg)
		c.StatusReporter.UpdateStatus(status, msg)
	}
}

// MetricRegistry returns the metrics registry associated with this context.
// This should be the metrics registry used by inputs to register their metrics.
// It's already registered to be published by the HTTP monitoring endpoint.
// If the context wasn't created by NewContext and its monitoring registry is
// nil, a new registry is created and returned.
func (c *Context) MetricRegistry() *monitoring.Registry {
	// It's a precaution in case the context wasn't created by NewContext.
	if c.monitoringRegistry == nil {
		c.monitoringRegistry = monitoring.NewRegistry()
	}

	return c.monitoringRegistry
}

// UnregisterMetrics removes the metrics registry from its parent registry and
// from the HTTP monitoring endpoint.
func (c *Context) UnregisterMetrics() {
	if c.monitoringRegistryCancel != nil {
		c.monitoringRegistryCancel()
	}
}

// NewPipelineClientListener returns a new beat.ClientListener which might be
// the PipelineClientListener or a beat.CombinedClientListener. It's the latter
// when clientListener is non-nil.
// The PipelineClientListener collects pipeline metrics for an input. The
// metrics are created on reg.
func NewPipelineClientListener(
	reg *monitoring.Registry,
	clientListener beat.ClientListener) beat.ClientListener {

	var pcl beat.ClientListener = &PipelineClientListener{
		eventsTotal: monitoring.NewUint(
			reg, "events_pipeline_total"),
		eventsFiltered: monitoring.NewUint(
			reg, "events_pipeline_filtered_total"),
		eventsPublished: monitoring.NewUint(
			reg, "events_pipeline_published_total"),
	}

	if clientListener != nil {
		pcl = &beat.CombinedClientListener{
			A: clientListener,
			B: pcl,
		}
	}
	return pcl
}

// PipelineClientListener implements beat.ClientListener to collect pipeline
// metrics per-input.
type PipelineClientListener struct {
	eventsTotal,
	eventsFiltered,
	eventsPublished *monitoring.Uint
}

func (i *PipelineClientListener) Closing() {
}

func (i *PipelineClientListener) Closed() {
}

func (i *PipelineClientListener) NewEvent() {
	i.eventsTotal.Inc()
}

func (i *PipelineClientListener) Filtered() {
	i.eventsFiltered.Inc()
}

func (i *PipelineClientListener) Published() {
	i.eventsPublished.Inc()
}

func (i *PipelineClientListener) DroppedOnPublish(beat.Event) {}

// TestContext provides the Input Test function with common environmental
// information and services.
type TestContext struct {
	// Logger provides a structured logger to inputs. The logger is initialized
	// with labels that will identify logs for the input.
	Logger *logp.Logger

	// Agent provides additional Beat info like instance ID or beat name.
	Agent beat.Info

	// Cancelation is used by Beats to signal the input to shut down.
	Cancelation Canceler
}

// Canceler is used to provide shutdown handling to the Context.
type Canceler interface {
	Done() <-chan struct{}
	Err() error
}

type cancelerCtx struct {
	Canceler
}

func GoContextFromCanceler(c Canceler) context.Context {
	return cancelerCtx{c}
}

func (c cancelerCtx) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (c cancelerCtx) Value(_ any) any {
	return nil
}
