// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpendpoint

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type HttpServer struct {
	server *http.Server
	ctx    context.Context
	stop   context.CancelFunc
	done   chan struct{}
}

func (h *HttpServer) Start() error {
	go func() {
		if h.server.TLSConfig != nil {
			logp.Info("Starting HTTPS server on %s", h.server.Addr)
			//certificate is already loaded. That's why the parameters are empty
			err := h.server.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				logp.Critical("Unable to start HTTPS server due to error: %v", err)
			}
		} else {
			logp.Info("Starting HTTP server on %s", h.server.Addr)
			err := h.server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logp.Critical("Unable to start HTTP server due to error: %v", err)
			}
		}
	}()

	return nil
}

func (h *HttpServer) Stop() {
	logp.Info("Stopping HTTP server")
	close(h.done)
	h.stop()
	h.server.Shutdown(h.ctx)
}

func createServer(in *HttpEndpoint) (*HttpServer, error) {
	server := &http.Server{
		Addr: in.config.ListenAddress + ":" + in.config.ListenPort,
	}

	http.HandleFunc(in.config.URL, in.apiResponse)
	tlsConfig, err := tlscommon.LoadTLSServerConfig(in.config.TLS)

	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		server.TLSConfig = tlsConfig.BuildModuleConfig(in.config.ListenAddress)
		if !in.config.ClientAuth {
			server.TLSConfig.ClientAuth = tls.NoClientCert
		}
	}

	ctx, cancel := context.WithCancel(context.TODO())
	h := &HttpServer{
		done: make(chan struct{}),
		ctx:  ctx,
		stop: cancel,
	}
	fmt.Printf("%+v\n", defaultConfig)
	h.server = server

	return h, nil
}
