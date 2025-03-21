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
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
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
	pipelineClientListener   *PipelineClientListener
}

// NewContext creates a new context with a metrics registry populated with
// 'id: id' and 'input: inputType'. The registry is registered to be published
// by the HTTP monitoring endpoint. The metrics registry is created on
// parentRegistry, if it's not nil, or on a new unregistered registry.
func NewContext(
	id,
	idWithoutName,
	inputType string,
	agent beat.Info,
	cancelation Canceler,
	statusReporter status.StatusReporter,
	parentRegistry *monitoring.Registry,
	log *logp.Logger) Context {
	if parentRegistry == nil || id == "" {
		log.Warn("registering metrics for %s, id: %s, with empty parent registry or empty ID",
			inputType, id)
		parentRegistry = monitoring.NewRegistry()
	}

	metricsID := strings.ReplaceAll(id, ".", "_")
	reg := parentRegistry.GetRegistry(metricsID)
	if reg == nil {
		reg = parentRegistry.NewRegistry(metricsID)
	}

	// add the necessary information so the registry can be published by the
	// HTTP monitoring endpoint.
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewString(reg, "id").Set(id)

	// register to be published by the HTTP monitoring endpoint.
	err := inputmon.RegisterMetrics(metricsID, reg)
	if err != nil {
		log.Errorf("failed to register metrics for '%s', id: %s,: %v",
			inputType, id, err)
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
		"id", id,
		"registry_name", metricsID,
		"uuid", uid)

	unreg := func() {
		metricsLog.Infow("unregistering",
			"input_type", inputType,
			"id", id,
			"registry_name",
			metricsID, "uuid", uid)
		parentRegistry.Remove(metricsID)
		// it's safe to make this call even if registering them failed.
		inputmon.UnregisterMetrics(metricsID)
	}
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

// UpdateMetricRegistry overrides the `id` and `input` entries in the metrics
// registry and returns the registry. It exists for backwards compatibility as
// some inputs set a different input type/name than their name.
func (c *Context) UpdateMetricRegistry(id, inputType string) *monitoring.Registry {
	monitoring.NewString(c.MetricRegistry(), "input").Set(inputType)
	monitoring.NewString(c.MetricRegistry(), "id").Set(id)

	return c.MetricRegistry()
}

// UnregisterMetrics removes the metrics registry from its parent registry and
// from the HTTP monitoring endpoint.
func (c *Context) UnregisterMetrics() {
	if c.monitoringRegistryCancel != nil {
		c.monitoringRegistryCancel()
	}
}

func (c *Context) PipelineClientListener() *PipelineClientListener {
	if c.pipelineClientListener != nil {
		return c.pipelineClientListener
	}

	c.pipelineClientListener = &PipelineClientListener{
		eventsTotal: monitoring.NewUint(
			c.MetricRegistry(), "events_pipeline_total"),
		eventsFiltered: monitoring.NewUint(
			c.MetricRegistry(), "events_pipeline_filtered_total"),
		eventsPublished: monitoring.NewUint(
			c.MetricRegistry(), "events_pipeline_published_total"),
	}

	return c.pipelineClientListener
}

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
