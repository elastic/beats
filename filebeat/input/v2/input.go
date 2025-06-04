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

	metricEventOutputTotal           = "events_output_total"
	metricEventOutputAckedTotal      = "events_output_acked_total"
	metricEventOutputDroppedTotal    = "events_output_dropped_total"
	metricEventOutputDeadLetterTotal = "events_output_dead_letter_total"
	metricEventOutputDuplicateTotal  = "events_output_duplicate_events_total"
	metricEventOutputErrTooManyTotal = "events_output_err_too_many_total"
	metricEventOutputRetryableTotal  = "events_output_retryable_errors_total"
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

	// MetricsRegistry is the registry collecting metrics for the input using
	// this context.
	MetricsRegistry *monitoring.Registry
}

func (c *Context) UpdateStatus(status status.Status, msg string) {
	if c.StatusReporter != nil {
		c.Logger.Debugf("updating status, status: '%s', message: '%s'", status.String(), msg)
		c.StatusReporter.UpdateStatus(status, msg)
	}
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

// NewPipelineOutputListener returns a new OutputListener which implements
// beat.OutputListener
// The OutputListener collects output metrics for an input. If the registry is
// nil, the metrics will be added on a new, unregistered registry.
// If there is already a metric with the same name in the registry,
// the existing metric will be used.
func NewPipelineOutputListener(reg *monitoring.Registry) *OutputListener {
	rreg := reg
	if rreg == nil {
		rreg = monitoring.NewRegistry()
	}

	return &OutputListener{
		eventsTotal: monitoring.NewUint(
			rreg, metricEventOutputTotal),
		eventsAcked: monitoring.NewUint(
			rreg, metricEventOutputAckedTotal),
		eventsDropped: monitoring.NewUint(
			rreg, metricEventOutputDroppedTotal),
		eventsDeadLetter: monitoring.NewUint(
			rreg, metricEventOutputDeadLetterTotal),
		eventsDuplicateEvents: monitoring.NewUint(
			rreg, metricEventOutputDuplicateTotal),
		eventsErrTooMany: monitoring.NewUint(
			rreg, metricEventOutputErrTooManyTotal),
		eventsRetryableErrors: monitoring.NewUint(
			rreg, metricEventOutputRetryableTotal),
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

	clientListener := NewPipelineClientListener(reg)
	outputListener := NewPipelineOutputListener(reg)

	pipeConnector := pipetool.WithClientConfigEdit(pconnector,
		func(orig beat.ClientConfig) (beat.ClientConfig, error) {
			var cl beat.ClientListener = clientListener
			if orig.ClientListener != nil {
				cl = &beat.CombinedClientListener{
					A: orig.ClientListener,
					B: clientListener,
				}
			}

			var ol beat.OutputListener = outputListener
			if orig.OutputListener != nil {
				ol = &beat.CombinedOutputListener{
					A: orig.OutputListener,
					B: outputListener,
				}
			}

			orig.ClientListener = cl
			orig.OutputListener = ol
			return orig, nil
		})

	return reg, pipeConnector, func() {
		// Unregister the metrics when the input finishes running.
		defer inputmon.CancelMetricsRegistry(
			inputID, name, parent, log)
	}
}

type OutputListener struct {
	eventsAcked,
	eventsDeadLetter,
	eventsDropped,
	eventsDuplicateEvents,
	eventsErrTooMany,
	eventsRetryableErrors,
	eventsTotal *monitoring.Uint
}

func (o *OutputListener) Acked() {
	o.eventsAcked.Inc()
}

func (o *OutputListener) DeadLetter() {
	o.eventsDeadLetter.Inc()
}
func (o *OutputListener) Dropped() {
	o.eventsDropped.Inc()
}

func (o *OutputListener) DuplicateEvents() {
	o.eventsDuplicateEvents.Inc()
}

func (o *OutputListener) ErrTooMany() {
	o.eventsErrTooMany.Inc()
}

func (o *OutputListener) RetryableError() {
	o.eventsRetryableErrors.Inc()
}

func (o *OutputListener) NewEvent() {
	o.eventsTotal.Inc()
}

// PipelineClientListener implements beat.ClientListener to collect pipeline
// metrics per-input.
type PipelineClientListener struct {
	eventsTotal,
	eventsFiltered,
	eventsPublished *monitoring.Uint
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

func (i *PipelineClientListener) Closing()                    {}
func (i *PipelineClientListener) Closed()                     {}
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
