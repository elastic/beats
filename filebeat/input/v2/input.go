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

// This file was contributed to by generative AI

package v2

import (
	"context"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/go-concert/unison"
)

const (
	metricEventsPipelineTotal     = "events_pipeline_total"
	metricEventsPipelineFiltered  = "events_pipeline_filtered_total"
	metricEventsPipelinePublished = "events_pipeline_published_total"
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
	//
	// An input started with a Context parameter, ctx, must shut down when
	// <-ctx.Cancelation.Done() does not block or when ctx.Cancelation.Error()
	// returns a non-nil error. If an input is started with a specific Context
	// it should only shut down in response to that Context's Cancelation.
	Cancelation Canceler

	// statusReporter provides a method to update the status of the underlying unit
	// that maps to the config. Note: Under standalone execution of Filebeat this is
	// expected to be nil.
	// Context implements the status.StatusReporter interface using this
	// statusReporter.
	statusReporter status.StatusReporter

	// MetricsRegistry is the registry collecting metrics for the input using
	// this context.
	MetricsRegistry *monitoring.Registry
}

// UpdateStatus Updates the status of this unit. This method is safe to use
// without a StatusReporter set.
func (c Context) UpdateStatus(status status.Status, msg string) {
	if c.statusReporter != nil {
		c.Logger.Debugf("updating status, status: '%s', message: '%s'", status.String(), msg)
		c.statusReporter.UpdateStatus(status, msg)
	}
}

// WithStatusReporter returns a copy of this context with the StatusReporter set
// to reporter.
func (c Context) WithStatusReporter(reporter status.StatusReporter) Context {
	c.statusReporter = reporter
	return c
}

// MetricsRegistryOverrideID sets the "id" variable in the Context's
// MetricsRegistry and returns the modified registry. This is required as some
// inputs do not use their input ID as the identifier for their metrics.
func MetricsRegistryOverrideID(reg *monitoring.Registry, id string) {
	monitoring.NewString(reg, inputmon.MetricKeyID).Set(id)
}

// MetricsRegistryOverrideInput sets the "input" variable in the Context's
// MetricsRegistry and returns the modified registry. This is required as some
// inputs do not use their input name for their metrics.
func MetricsRegistryOverrideInput(reg *monitoring.Registry, inputName string) {
	monitoring.NewString(reg, inputmon.MetricKeyInput).Set(inputName)
}

// NewPipelineClientListener returns a new beat.ClientListener.
// The PipelineClientListener collects pipeline metrics for an input. The
// metrics are created on reg.
func NewPipelineClientListener(reg *monitoring.Registry) *PipelineClientListener {
	return &PipelineClientListener{
		eventsTotal:     monitoring.NewUint(reg, metricEventsPipelineTotal),
		eventsFiltered:  monitoring.NewUint(reg, metricEventsPipelineFiltered),
		eventsPublished: monitoring.NewUint(reg, metricEventsPipelinePublished),
	}
}

// PrepareInputMetrics creates a new monitoring.Registry on parent for the given
// inputID and a PipelineClientListener using the new monitoring.Registry.
// Then it wrappers the given beat.PipelineConnector to add the newly created
// PipelineClientListener to the beat.ClientConfig.
//
// It returns the new monitoring.Registry and the wrapped beat.PipelineConnector
// and a function to unregister the new monitoring.Registry from parent.
func PrepareInputMetrics(
	inputID,
	name string,
	parent *monitoring.Registry,
	pconnector beat.PipelineConnector,
	log *logp.Logger) (*monitoring.Registry, beat.PipelineConnector, func()) {

	reg := inputmon.NewMetricsRegistry(
		inputID, name, parent, log)
	listener := NewPipelineClientListener(reg)

	pc := pipetool.WithClientConfigEdit(pconnector,
		func(orig beat.ClientConfig) (beat.ClientConfig, error) {
			var pcl beat.ClientListener = listener
			if orig.ClientListener != nil {
				pcl = &beat.CombinedClientListener{
					A: orig.ClientListener,
					B: listener,
				}
			}

			orig.ClientListener = pcl
			return orig, nil
		})

	return reg, pc, func() {
		// Unregister the metrics when the input finishes running.
		defer inputmon.CancelMetricsRegistry(
			inputID, name, parent, log)
	}
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
	//
	// An input started with a Context parameter, ctx, must shut down when
	// <-ctx.Cancelation.Done() does not block or when ctx.Cancelation.Error()
	// returns a non-nil error. If an input is started with a specific Context
	// it should only shut down in response to that Context's Cancelation.
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
