// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/docker/docker/daemon/logger"

	"github.com/elastic/beats/v8/x-pack/dockerlogbeat/pipelinemanager"

	"github.com/docker/docker/pkg/ioutils"
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

// capabilitiesResponse represents the response to a capabilities request
type capabilitiesResponse struct {
	Err string
	Cap logger.Capability
}

// logsRequest represents the request object we get from a `docker logs` call
type logsRequest struct {
	Info   logger.Info
	Config logger.ReadConfig
}

func reportCaps() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&capabilitiesResponse{
			Cap: logger.Capability{ReadLogs: true},
		})
	}
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

		pm.Logger.Debugf("Got start request object from container %#v\n", startReq.Info.ContainerName)
		pm.Logger.Debugf("Got a container with the following labels: %#v\n", startReq.Info.ContainerLabels)
		pm.Logger.Debugf("Got a container with the following log opts: %#v\n", startReq.Info.Config)

		cfg, err := pipelinemanager.NewCfgFromRaw(startReq.Info.Config)
		if err != nil {
			http.Error(w, errors.Wrap(err, "error creating client config").Error(), http.StatusBadRequest)
			return
		}
		pm.Logger.Debugf("Got config: %#v", cfg)
		cl, err := pm.CreateClientWithConfig(cfg, startReq.Info, startReq.File)
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
		pm.Logger.Debugf("Got stop request object %#v\n", stopReq)
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

func readLogHandler(pm *pipelinemanager.PipelineManager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var logReq logsRequest
		err := json.NewDecoder(r.Body).Decode(&logReq)
		if err != nil {
			http.Error(w, errors.Wrap(err, "error decoding json request").Error(), http.StatusBadRequest)
			return
		}

		pm.Logger.Debugf("Got logging request for container %s\n", logReq.Info.ContainerName)
		stream, err := pm.CreateReaderForContainer(logReq.Info, logReq.Config)
		if err != nil {
			http.Error(w, errors.Wrap(err, "error creating log reader").Error(), http.StatusBadRequest)
			return
		}
		defer stream.Close()
		w.Header().Set("Content-Type", "application/x-json-stream")
		wf := ioutils.NewWriteFlusher(w)
		defer wf.Close()
		io.Copy(wf, stream)

	} //end func
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
