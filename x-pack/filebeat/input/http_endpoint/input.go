// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/go-concert/ctxtool"
)

const (
	inputName = "http_endpoint"
)

type httpEndpoint struct {
	config    config
	addr      string
	tlsConfig *tls.Config
}

func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Manager:    v2.ConfigureWith(configure),
	}
}

func configure(cfg *conf.C) (v2.Input, error) {
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return newHTTPEndpoint(conf)
}

func newHTTPEndpoint(config config) (*httpEndpoint, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%v:%v", config.ListenAddress, config.ListenPort)

	var tlsConfig *tls.Config
	tlsConfigBuilder, err := tlscommon.LoadTLSServerConfig(config.TLS)
	if err != nil {
		return nil, err
	}
	if tlsConfigBuilder != nil {
		tlsConfig = tlsConfigBuilder.BuildServerConfig(addr)
	}

	return &httpEndpoint{
		config:    config,
		tlsConfig: tlsConfig,
		addr:      addr,
	}, nil
}

func (*httpEndpoint) Name() string { return inputName }

func (e *httpEndpoint) Test(_ v2.TestContext) error {
	l, err := net.Listen("tcp", e.addr)
	if err != nil {
		return err
	}
	return l.Close()
}

func (e *httpEndpoint) Run(ctx v2.Context, pipeline beat.Pipeline) error {
	metrics := newInputMetrics(ctx.ID)
	defer metrics.Close()

	if e.config.Tracer != nil {
		id := sanitizeFileName(ctx.ID)
		e.config.Tracer.Filename = strings.ReplaceAll(e.config.Tracer.Filename, "*", id)
	}

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		EventListener: newEventACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	err = servers.serve(ctx, e, client.Publish, metrics)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("unable to start server due to error: %w", err)
	}
	return nil
}

// sanitizeFileName returns name with ":" and "/" replaced with "_", removing repeated instances.
// The request.tracer.filename may have ":" when a http_endpoint input has cursor config and
// the macOS Finder will treat this as path-separator and causes to show up strange filepaths.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, ":", string(filepath.Separator))
	name = filepath.Clean(name)
	return strings.ReplaceAll(name, string(filepath.Separator), "_")
}

// servers is the package-level server pool.
var servers = pool{servers: make(map[string]*server)}

// pool is a concurrence-safe pool of http servers.
type pool struct {
	mu sync.Mutex
	// servers is the server pool keyed on their address/port.
	servers map[string]*server
}

// serve runs an http server configured with the provided end-point and
// publishing to pub. The server will run until either the context is
// cancelled or the context of another end-point sharing the same address
// has had its context cancelled. If an end-point is re-registered with
// the same address and mux pattern, serve will return an error.
func (p *pool) serve(ctx v2.Context, e *httpEndpoint, pub func(beat.Event), metrics *inputMetrics) error {
	log := ctx.Logger.With("address", e.addr)
	pattern := e.config.URL

	u, err := url.Parse(pattern)
	if err != nil {
		return err
	}
	metrics.route.Set(u.Path)
	metrics.isTLS.Set(e.tlsConfig != nil)

	var prg *program
	if e.config.Program != "" {
		prg, err = newProgram(e.config.Program, log)
		if err != nil {
			return err
		}
	}

	p.mu.Lock()
	s, ok := p.servers[e.addr]
	if ok {
		err = checkTLSConsistency(e.addr, s.tls, e.config.TLS)
		if err != nil {
			p.mu.Unlock()
			return err
		}

		if old, ok := s.idOf[pattern]; ok {
			err = fmt.Errorf("pattern already exists for %s: %s old=%s new=%s",
				e.addr, pattern, old, ctx.ID)
			s.setErr(err)
			s.cancel()
			p.mu.Unlock()
			return err
		}
		log.Infof("Adding %s end point to server on %s", pattern, e.addr)
		s.mux.Handle(pattern, newHandler(s.ctx, e.config, prg, pub, log, metrics))
		s.idOf[pattern] = ctx.ID
		p.mu.Unlock()
		<-s.ctx.Done()
		return s.getErr()
	}

	mux := http.NewServeMux()
	srv := &http.Server{Addr: e.addr, TLSConfig: e.tlsConfig, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	s = &server{
		idOf: map[string]string{pattern: ctx.ID},
		tls:  e.config.TLS,
		mux:  mux,
		srv:  srv,
	}
	s.ctx, s.cancel = ctxtool.WithFunc(ctx.Cancelation, func() { srv.Close() })
	mux.Handle(pattern, newHandler(s.ctx, e.config, prg, pub, log, metrics))
	p.servers[e.addr] = s
	p.mu.Unlock()

	if e.tlsConfig != nil {
		log.Infof("Starting HTTPS server on %s with %s end point", srv.Addr, pattern)
		// The certificate is already loaded so we do not need
		// to pass the cert file and key file parameters.
		err = listenAndServeTLS(s.srv, "", "", metrics)
	} else {
		log.Infof("Starting HTTP server on %s with %s end point", srv.Addr, pattern)
		err = listenAndServe(s.srv, metrics)
	}
	p.mu.Lock()
	delete(p.servers, e.addr)
	p.mu.Unlock()
	s.setErr(err)
	s.cancel()
	return err
}

func listenAndServeTLS(srv *http.Server, certFile, keyFile string, metrics *inputMetrics) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	metrics.bindAddr.Set(ln.Addr().String())

	defer ln.Close()

	return srv.ServeTLS(ln, certFile, keyFile)
}

func listenAndServe(srv *http.Server, metrics *inputMetrics) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	metrics.bindAddr.Set(ln.Addr().String())
	return srv.Serve(ln)
}

func checkTLSConsistency(addr string, old, new *tlscommon.ServerConfig) error {
	if old == nil && new == nil {
		return nil
	}
	if (old == nil) != (new == nil) {
		return invalidTLSStateErr{addr: addr, reason: "mixed TLS and unencrypted", old: old, new: new}
	}
	if !reflect.DeepEqual(old, new) {
		return invalidTLSStateErr{addr: addr, reason: "configuration options do not agree", old: old, new: new}
	}
	return nil
}

type invalidTLSStateErr struct {
	addr     string
	reason   string
	old, new *tlscommon.ServerConfig
}

func (e invalidTLSStateErr) Error() string {
	if e.old == nil || e.new == nil {
		return fmt.Sprintf("inconsistent TLS usage on %s: %s", e.addr, e.reason)
	}
	return fmt.Sprintf("inconsistent TLS configuration on %s: %s: old=%s new=%s",
		e.addr, e.reason, renderTLSConfig(e.old), renderTLSConfig(e.new))
}

func renderTLSConfig(tls *tlscommon.ServerConfig) string {
	c, err := conf.NewConfigFrom(tls)
	if err != nil {
		return fmt.Sprintf("!%v", err)
	}
	var m mapstr.M
	err = c.Unpack(&m)
	if err != nil {
		return fmt.Sprintf("!%v", err)
	}
	return m.String()
}

// server is a collection of http end-points sharing the same underlying
// http.Server.
type server struct {
	// idOf is a map of mux pattern
	// to input IDs for the server.
	idOf map[string]string

	tls *tlscommon.ServerConfig

	mux *http.ServeMux
	srv *http.Server

	ctx    context.Context
	cancel func()

	mu  sync.Mutex
	err error
}

func (s *server) setErr(err error) {
	s.mu.Lock()
	s.err = err
	s.mu.Unlock()
}

func (s *server) getErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func newHandler(ctx context.Context, c config, prg *program, pub func(beat.Event), log *logp.Logger, metrics *inputMetrics) http.Handler {
	h := &handler{
		ctx:      ctx,
		log:      log,
		txBaseID: newID(),

		publish: pub,
		metrics: metrics,
		validator: apiValidator{
			basicAuth:    c.BasicAuth,
			username:     c.Username,
			password:     c.Password,
			method:       c.Method,
			contentType:  c.ContentType,
			secretHeader: c.SecretHeader,
			secretValue:  c.SecretValue,
			hmacHeader:   c.HMACHeader,
			hmacKey:      c.HMACKey,
			hmacType:     c.HMACType,
			hmacPrefix:   c.HMACPrefix,
		},
		program:               prg,
		messageField:          c.Prefix,
		responseCode:          c.ResponseCode,
		responseBody:          htmlEscape(c.ResponseBody),
		includeHeaders:        canonicalizeHeaders(c.IncludeHeaders),
		preserveOriginalEvent: c.PreserveOriginalEvent,
		crc:                   newCRC(c.CRCProvider, c.CRCSecret),
	}
	if c.Tracer.enabled() {
		w := zapcore.AddSync(c.Tracer)
		go func() {
			// Close the logger when we are done.
			<-ctx.Done()
			c.Tracer.Close()
		}()
		core := ecszap.NewCore(
			ecszap.NewDefaultEncoderConfig(),
			w,
			zap.DebugLevel,
		)
		h.reqLogger = zap.New(core)
		h.host = c.ListenAddress + ":" + c.ListenPort
		if c.TLS != nil && c.TLS.IsEnabled() {
			h.scheme = "https"
		} else {
			h.scheme = "http"
		}
	} else if c.Tracer != nil {
		// We have a trace log name, but we are not enabled,
		// so remove all trace logs we own.
		err := os.Remove(c.Tracer.Filename)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Errorw("failed to remove request trace log", "path", c.Tracer.Filename, "error", err)
		}
		ext := filepath.Ext(c.Tracer.Filename)
		base := strings.TrimSuffix(c.Tracer.Filename, ext)
		paths, err := filepath.Glob(base + "-" + lumberjackTimestamp + ext)
		if err != nil {
			log.Errorw("failed to collect request trace log path names", "error", err)
		}
		for _, p := range paths {
			err = os.Remove(p)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				log.Errorw("failed to remove request trace log", "path", p, "error", err)
			}
		}
	}
	return h
}

// lumberjackTimestamp is a glob expression matching the time format string used
// by lumberjack when rolling over logs, "2006-01-02T15-04-05.000".
// https://github.com/natefinch/lumberjack/blob/4cb27fcfbb0f35cb48c542c5ea80b7c1d18933d0/lumberjack.go#L39
const lumberjackTimestamp = "[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]-[0-9][0-9]-[0-9][0-9].[0-9][0-9][0-9]"

func htmlEscape(s string) string {
	var buf bytes.Buffer
	json.HTMLEscape(&buf, []byte(s))
	return buf.String()
}

// newID returns an ID derived from the current time.
func newID() string {
	var data [8]byte
	binary.LittleEndian.PutUint64(data[:], uint64(time.Now().UnixNano()))
	return base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(data[:])
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	bindAddr            *monitoring.String // bind address of input
	route               *monitoring.String // request route
	isTLS               *monitoring.Bool   // whether the input is listening on a TLS connection
	apiErrors           *monitoring.Uint   // number of API errors
	batchesReceived     *monitoring.Uint   // number of event arrays received
	batchesPublished    *monitoring.Uint   // number of event arrays published
	batchesACKedTotal   *monitoring.Uint   // Number of event arrays ACKed.
	eventsPublished     *monitoring.Uint   // number of events published
	contentLength       metrics.Sample     // histogram of request content lengths.
	batchSize           metrics.Sample     // histogram of the received batch sizes.
	batchProcessingTime metrics.Sample     // histogram of the elapsed successful batch processing times in nanoseconds (time of handler start to time of ACK for non-empty batches).
	batchACKTime        metrics.Sample     // histogram of the elapsed successful batch acking times in nanoseconds (time of handler start to time of ACK for non-empty batches).
}

func newInputMetrics(id string) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, nil)
	out := &inputMetrics{
		unregister:          unreg,
		bindAddr:            monitoring.NewString(reg, "bind_address"),
		route:               monitoring.NewString(reg, "route"),
		isTLS:               monitoring.NewBool(reg, "is_tls_connection"),
		apiErrors:           monitoring.NewUint(reg, "api_errors_total"),
		batchesReceived:     monitoring.NewUint(reg, "batches_received_total"),
		batchesPublished:    monitoring.NewUint(reg, "batches_published_total"),
		batchesACKedTotal:   monitoring.NewUint(reg, "batches_acked_total"),
		eventsPublished:     monitoring.NewUint(reg, "events_published_total"),
		contentLength:       metrics.NewUniformSample(1024),
		batchSize:           metrics.NewUniformSample(1024),
		batchProcessingTime: metrics.NewUniformSample(1024),
		batchACKTime:        metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "size", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.contentLength))
	_ = adapter.NewGoMetrics(reg, "batch_size", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchSize))
	_ = adapter.NewGoMetrics(reg, "batch_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime))
	_ = adapter.NewGoMetrics(reg, "batch_ack_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchACKTime))

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}
