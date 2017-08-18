package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/mb"
)

type HttpServer struct {
	server     *http.Server
	ctx        context.Context
	stop       context.CancelFunc
	done       chan struct{}
	eventQueue chan server.Event
}

type HttpEvent struct {
	event common.MapStr
	meta  server.Meta
}

func (h *HttpEvent) GetEvent() common.MapStr {
	return h.event
}

func (h *HttpEvent) GetMeta() server.Meta {
	return h.meta
}

func NewHttpServer(mb mb.BaseMetricSet) (server.Server, error) {
	config := defaultHttpConfig()
	err := mb.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	h := &HttpServer{
		done:       make(chan struct{}),
		eventQueue: make(chan server.Event),
		ctx:        ctx,
		stop:       cancel,
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler: http.HandlerFunc(h.handleFunc),
	}
	h.server = httpServer

	return h, nil
}

func (h *HttpServer) Start() error {
	go func() {

		logp.Info("Starting http server on %s", h.server.Addr)
		err := h.server.ListenAndServe()
		if err != nil {
			logp.Critical("Unable to start HTTP server due to error: %v", err)
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
			"path": req.URL.String(),
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

		payload := common.MapStr{
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
		writer.Write([]byte("HTTP Server accepts data via POST"))
	}
}
