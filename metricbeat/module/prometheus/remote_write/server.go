package remote_write

import (
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/mb"
	"net"
	"net/http"
	"strconv"
)

type HttpProtoServer struct {
	server     *http.Server
	ctx        context.Context
	stop       context.CancelFunc
	done       chan struct{}
	eventQueue chan server.Event
}


func NewHttpServer(mb mb.BaseMetricSet) (server.Server, error) {
	config := defaultHttpConfig()
	err := mb.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSServerConfig(config.TLS)
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
		Addr:    net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Handler: http.HandlerFunc(h.handleFunc),
	}
	if tlsConfig != nil {
		httpServer.TLSConfig = tlsConfig.BuildModuleConfig(config.Host)
	}
	h.server = httpServer

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
