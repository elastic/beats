// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/docker/docker/daemon/logger"

	"github.com/elastic/beats/v7/x-pack/dockerlogbeat/pipelinemanager"

	"github.com/pkg/errors"
)

// StartLoggingRequest represents the request object we get on a call to //LogDriver.StartLogging
type StartLoggingRequest struct {
	File string
	Info logger.Info
}

// StopLoggingRequest represents the request object we get on a call to //LogDriver.StopLogging
type StopLoggingRequest struct {
	File string
}

// This gets called when a container starts that requests the log driver
func startLoggingHandler(pm *pipelinemanager.PipelineManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var startReq StartLoggingRequest
		err := json.NewDecoder(r.Body).Decode(&startReq)
		if err != nil {
			http.Error(w, errors.Wrap(err, "error decoding json request").Error(), http.StatusBadRequest)
			return
		}

		pm.Logger.Debugf("Homepath: %v\n", filepath.Dir(os.Args[0]))
		pm.Logger.Infof("Got start request object from container %#v\n", startReq.Info.ContainerName)
		pm.Logger.Debugf("Got a container with the following labels: %#v\n", startReq.Info.ContainerLabels)
		pm.Logger.Debugf("Got a container with the following log opts: %#v\n", startReq.Info.Config)

		cl, err := pm.CreateClientWithConfig(startReq.Info, startReq.File)
		if err != nil {
			http.Error(w, errors.Wrap(err, "error creating client").Error(), http.StatusBadRequest)
			return
		}

		go cl.ConsumePipelineAndSend()

		respondOK(w)
	} // end func
}

// This gets called when a container using the log driver stops
func stopLoggingHandler(pm *pipelinemanager.PipelineManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var stopReq StopLoggingRequest
		err := json.NewDecoder(r.Body).Decode(&stopReq)
		if err != nil {
			http.Error(w, errors.Wrap(err, "error decoding json request").Error(), http.StatusBadRequest)
			return
		}
		pm.Logger.Infof("Got stop request object %#v\n", stopReq)
		// Run the stop async, since nothing 'depends' on it,
		// and we can break people's docker automation if this times out.
		go func() {
			err = pm.CloseClientWithFile(stopReq.File)
			if err != nil {
				pm.Logger.Errorf(" Got stop request error %#v\n", err)
			}
		}()

		respondOK(w)
	} // end func
}

// For the start/stop handler, the daemon expects back an error object. If the body is empty, then all is well.
func respondOK(w http.ResponseWriter) {
	res := struct {
		Err string
	}{
		"",
	}

	json.NewEncoder(w).Encode(&res)
}
