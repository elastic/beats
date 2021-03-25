// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
	"github.com/gorilla/mux"
)

// New creates a new server exposing metrics and process information.
func New(
	log *logger.Logger,
	endpointConfig api.Config,
	ns func(string) *monitoring.Namespace,
	routesFetchFn func() *sorted.Set,
) (*api.Server, error) {
	if err := createAgentMonitoringDrop(endpointConfig.Host); err != nil {
		// log but ignore
		log.Errorf("failed to create monitoring drop: %v", err)
	}

	cfg, err := common.NewConfigFrom(endpointConfig)
	if err != nil {
		return nil, err
	}

	return exposeMetricsEndpoint(log, cfg, ns, routesFetchFn)
}

func exposeMetricsEndpoint(log *logger.Logger, config *common.Config, ns func(string) *monitoring.Namespace, routesFetchFn func() *sorted.Set) (*api.Server, error) {
	r := mux.NewRouter()
	r.HandleFunc("/stats", statsHandler(ns("stats")))
	r.HandleFunc("/processes", processesHandler(routesFetchFn))
	r.HandleFunc("/processes/{processID}", processHandler())

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
