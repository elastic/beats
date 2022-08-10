// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/mux"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/monitoring/report/buffer"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
)

// New creates a new server exposing metrics and process information.
func New(
	log *logger.Logger,
	endpointConfig api.Config,
	ns func(string) *monitoring.Namespace,
	routesFetchFn func() *sorted.Set,
	enableProcessStats bool,
	enableBuffer bool,
) (*api.Server, error) {
	if err := createAgentMonitoringDrop(endpointConfig.Host); err != nil {
		// log but ignore
		log.Errorf("failed to create monitoring drop: %v", err)
	}

	cfg, err := common.NewConfigFrom(endpointConfig)
	if err != nil {
		return nil, err
	}

	return exposeMetricsEndpoint(log, cfg, ns, routesFetchFn, enableProcessStats, enableBuffer)
}

func exposeMetricsEndpoint(log *logger.Logger, config *common.Config, ns func(string) *monitoring.Namespace, routesFetchFn func() *sorted.Set, enableProcessStats bool, enableBuffer bool) (*api.Server, error) {
	r := mux.NewRouter()
	statsHandler := statsHandler(ns("stats"))
	r.Handle("/stats", createHandler(statsHandler))

	if enableProcessStats {
		r.HandleFunc("/processes", processesHandler(routesFetchFn))
		r.Handle("/processes/{processID}", createHandler(processHandler(statsHandler)))
		r.Handle("/processes/{processID}/", createHandler(processHandler(statsHandler)))
		r.Handle("/processes/{processID}/{beatsPath}", createHandler(processHandler(statsHandler)))
	}

	if enableBuffer {
		bufferReporter, err := buffer.MakeReporter(beat.Info{}, config) // beat.Info is not used by buffer reporter
		if err != nil {
			return nil, fmt.Errorf("unable to create buffer reporter for elastic-agent: %w", err)
		}
		r.Handle("/buffer", bufferReporter)
	}

	mux := http.NewServeMux()
	mux.Handle("/", r)

	return api.New(log, mux, config)
}

func createAgentMonitoringDrop(drop string) error {
	if drop == "" || runtime.GOOS == "windows" {
		return nil
	}

	path := strings.TrimPrefix(drop, "unix://")
	if strings.HasSuffix(path, ".sock") {
		path = filepath.Dir(path)
	}

	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		// create
		if err := os.MkdirAll(path, 0775); err != nil {
			return err
		}
	}

	return os.Chown(path, os.Geteuid(), os.Getegid())
}

func errorWithStatus(status int, err error) *statusError {
	return &statusError{
		err:    err,
		status: status,
	}
}

func errorfWithStatus(status int, msg string, args ...string) *statusError {
	err := fmt.Errorf(msg, args)
	return errorWithStatus(status, err)
}

// StatusError holds correlation between error and a status
type statusError struct {
	err    error
	status int
}

func (s *statusError) Status() int {
	return s.status
}

func (s *statusError) Error() string {
	return s.err.Error()
}
