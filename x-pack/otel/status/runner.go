// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/config"

	"github.com/gohugoio/hashstructure"
)

type statusFactory struct {
	factory  cfgfile.RunnerFactory
	reporter RunnerReporter
}

// runnerRemover is implemented by the status reporters of a group reporter,
// to delete the runner's state from the group once the runner stopped.
type runnerRemover interface {
	remove()
}

func (s *statusFactory) Create(p beat.PipelineConnector, config *config.C) (cfgfile.Runner, error) {
	runner, err := s.factory.Create(p, config)
	if err != nil {
		return nil, err
	}
	runnerWithStatus, ok := runner.(status.WithStatusReporter)
	if !ok {
		return runner, nil
	}
	inputId, err := getInputId(config)
	if err != nil {
		return nil, err
	}
	statusReporter := s.reporter.GetReporterForRunner(inputId)
	runnerWithStatus.SetStatusReporter(statusReporter)
	if remover, ok := statusReporter.(runnerRemover); ok {
		return wrapRunner(runner, runnerWithStatus, remover.remove), nil
	}
	return runner, nil
}

// Unwrap returns the wrapped factory, so callers can use its optional
// interfaces, such as Heartbeat's config hash function.
func (s *statusFactory) Unwrap() cfgfile.RunnerFactory {
	return s.factory
}

func (s *statusFactory) CheckConfig(config *config.C) error {
	return s.factory.CheckConfig(config)
}

func StatusReporterFactory(reporter RunnerReporter) cfgfile.FactoryWrapper {
	return func(f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
		return &statusFactory{factory: f, reporter: reporter}
	}
}

// statusRunner removes the status of the runner it wraps from its group once
// the runner stopped, as not every runner reports that it stopped.
type statusRunner struct {
	cfgfile.Runner
	// withStatus is the wrapped runner, only runners that accept a status
	// reporter are wrapped.
	withStatus status.WithStatusReporter
	remove     func()
}

func (r *statusRunner) Stop() {
	r.Runner.Stop()
	r.remove()
}

// Unwrap returns the wrapped runner, so callers can use its concrete type.
func (r *statusRunner) Unwrap() cfgfile.Runner {
	return r.Runner
}

func (r *statusRunner) SetStatusReporter(reporter status.StatusReporter) {
	r.withStatus.SetStatusReporter(reporter)
}

// updatableRunner is implemented by runners whose config can be updated in
// place, such as Heartbeat monitors.
type updatableRunner interface {
	Update(newConfig *config.C) error
}

// wrapRunner wraps runner in a statusRunner that keeps the optional interfaces
// the runner implements, as callers check for them.
func wrapRunner(runner cfgfile.Runner, withStatus status.WithStatusReporter, remove func()) cfgfile.Runner {
	wrapped := &statusRunner{Runner: runner, withStatus: withStatus, remove: remove}
	diag, isDiag := runner.(diagnostics.DiagnosticReporter)
	updatable, isUpdatable := runner.(updatableRunner)
	switch {
	case isDiag && isUpdatable:
		return &struct {
			*statusRunner
			diagnostics.DiagnosticReporter
			updatableRunner
		}{wrapped, diag, updatable}
	case isDiag:
		return &struct {
			*statusRunner
			diagnostics.DiagnosticReporter
		}{wrapped, diag}
	case isUpdatable:
		return &struct {
			*statusRunner
			updatableRunner
		}{wrapped, updatable}
	default:
		return wrapped
	}
}

func getInputId(config *config.C) (string, error) {
	var h map[string]any
	err := config.Unpack(&h)
	if err != nil {
		return "", fmt.Errorf("could not unpack config: %w", err)
	}
	if inputId, ok := h["id"]; ok {
		return fmt.Sprintf("%v", inputId), nil
	}

	inputId, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute id from configuration: %w", err)
	}

	return fmt.Sprintf("%x", inputId), nil
}
