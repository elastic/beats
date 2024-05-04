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

package http

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"github.com/elastic/beats/v7/metricbeat/helper/server"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type HttpServer struct {
	server     *http.Server
	ctx        context.Context
	stop       context.CancelFunc
	done       chan struct{}
	eventQueue chan server.Event
}

type HttpEvent struct {
	event mapstr.M
	meta  server.Meta
}

func (h *HttpEvent) GetEvent() mapstr.M {
	return h.event
}

func (h *HttpEvent) GetMeta() server.Meta {
	return h.meta
}

func getDefaultHttpServer(mb mb.BaseMetricSet) (*HttpServer, error) {
	config := defaultHttpConfig()
	err := mb.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSServerConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.TODO())
	h := &HttpServer{
		done:       make(chan struct{}),
		eventQueue: make(chan server.Event),
		ctx:        ctx,
		stop:       cancel,
	}

	httpServer := &http.Server{
		Addr: net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
	}
	if tlsConfig != nil {
		httpServer.TLSConfig = tlsConfig.BuildServerConfig(config.Host)
	}
	h.server = httpServer
	return h, nil
}

func NewHttpServer(mb mb.BaseMetricSet) (server.Server, error) {
	h, err := getDefaultHttpServer(mb)
	if err != nil {
		return nil, err
	}
	h.server.Handler = http.HandlerFunc(h.handleFunc)

	return h, nil
}

func NewHttpServerWithHandler(mb mb.BaseMetricSet, handlerFunc http.HandlerFunc) (server.Server, error) {
	h, err := getDefaultHttpServer(mb)
	if err != nil {
		return nil, err
	}
	h.server.Handler = handlerFunc

	return h, nil
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
	close(h.done)
	h.stop()
	h.server.Shutdown(h.ctx)
	close(h.eventQueue)
}

func (h *HttpServer) GetEvents() chan server.Event {
	return h.eventQueue
}

func (h *HttpServer) handleFunc(writer http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		meta := server.Meta{
			"path":    req.URL.String(),
			"address": req.RemoteAddr,
		}

		contentType := req.Header.Get("Content-Type")
		if contentType != "" {
			meta["Content-Type"] = contentType
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			logp.Err("Error reading body: %v", err)
			http.Error(writer, "Unexpected error reading request payload", http.StatusBadRequest)
			return
		}

		payload := mapstr.M{
			server.EventDataKey: body,
		}

		event := &HttpEvent{
			event: payload,
			meta:  meta,
		}
		h.eventQueue <- event
		writer.WriteHeader(http.StatusAccepted)

	case "GET":
		writer.WriteHeader(http.StatusOK)
		if req.TLS != nil {
			writer.Write([]byte("HTTPS Server accepts data via POST"))
		} else {
			writer.Write([]byte("HTTP Server accepts data via POST"))
		}

	}
}
