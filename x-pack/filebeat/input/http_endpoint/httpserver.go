// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"context"
	"net/http"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type HttpServer struct {
	log    *logp.Logger
	server *http.Server
	ctx    context.Context
	stop   context.CancelFunc
}

func (h *HttpServer) Start() {
	go func() {
		if h.server.TLSConfig != nil {
			h.log.Infof("Starting HTTPS server on %s", h.server.Addr)
			//certificate is already loaded. That's why the parameters are empty
			err := h.server.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				h.log.Fatalf("Unable to start HTTPS server due to error: %v", err)
			}
		} else {
			h.log.Infof("Starting HTTP server on %s", h.server.Addr)
			err := h.server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				h.log.Fatalf("Unable to start HTTP server due to error: %v", err)
			}
		}
	}()
}

func (h *HttpServer) Stop() {
	h.log.Info("Stopping HTTP server")
	h.stop()
	if err := h.server.Shutdown(h.ctx); err != nil {
		h.log.Fatalf("Unable to stop HTTP server due to error: %v", err)
	}
}

func createServer(in *HttpEndpoint) (*HttpServer, error) {
	mux := http.NewServeMux()
	responseHandler := http.HandlerFunc(in.apiResponse)
	mux.Handle(in.config.URL, in.validateRequest(responseHandler))
	server := &http.Server{
		Addr:    in.config.ListenAddress + ":" + in.config.ListenPort,
		Handler: mux,
	}

	tlsConfig, err := tlscommon.LoadTLSServerConfig(in.config.TLS)
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		server.TLSConfig = tlsConfig.BuildModuleConfig(in.config.ListenAddress)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	h := &HttpServer{
		ctx:  ctx,
		stop: cancel,
		log:  logp.NewLogger("http_server"),
	}
	h.server = server

	return h, nil
}
