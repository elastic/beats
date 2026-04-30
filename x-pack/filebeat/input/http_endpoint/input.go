// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

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
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlsutil"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	inputName = "http_endpoint"
)

type httpEndpoint struct {
	config    config
	addr      string
	tlsConfig *tls.Config
	logger    *logp.Logger
}

func Plugin(log *logp.Logger) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Manager:    v2.ConfigureWith(configure, log),
	}
}

func configure(cfg *conf.C, logger *logp.Logger) (v2.Input, error) {
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return newHTTPEndpoint(conf, logger)
}

func newHTTPEndpoint(config config, logger *logp.Logger) (*httpEndpoint, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(config.ListenAddress, config.ListenPort)

	var tlsConfig *tls.Config
	tlsConfigBuilder, err := tlscommon.LoadTLSServerConfig(config.TLS, logger)
	if err != nil {
		return nil, err
	}
	if tlsConfigBuilder != nil {
		tlsConfig = tlsConfigBuilder.BuildServerConfig(addr)
		if err := tlsutil.SetupCertReload(tlsConfig, config.TLS); err != nil {
			return nil, err
		}
	}

	return &httpEndpoint{
		config:    config,
		tlsConfig: tlsConfig,
		addr:      addr,
		logger:    logger,
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
	ctx.UpdateStatus(status.Starting, "")
	ctx.UpdateStatus(status.Configuring, "")

	metrics := newInputMetrics(ctx.MetricsRegistry, ctx.Logger)

	if e.config.Tracer.enabled() {
		id := sanitizeFileName(ctx.IDWithoutName)
		path := strings.ReplaceAll(e.config.Tracer.Filename, "*", id)
		resolved, ok, err := httplog.ResolvePathInLogsFor(inputName, path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("request tracer path %q must be within %q path", path, paths.Resolve(paths.Logs, inputName))
		}
		e.config.Tracer.Filename = resolved
	}

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		EventListener: newEventACKHandler(),
	})
	if err != nil {
		ctx.UpdateStatus(status.Failed, "failed to create pipeline client: "+err.Error())
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	err = servers.serve(ctx, e, client.Publish, metrics)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("unable to start server due to error: %w", err)
	}
	ctx.UpdateStatus(status.Stopped, "")
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

// pool is a concurrent-safe pool of HTTP servers.
//
// Lock ordering: pool.mu must be acquired before [mux].mu (the
// read-write lock inside each [server]'s mux). Registration,
// deregistration, and listener-goroutine cleanup all acquire pool.mu
// first; mux.ServeHTTP acquires only mux.mu (read lock) and never
// pool.mu.
type pool struct {
	mu sync.Mutex
	// servers is the server pool keyed on their address/port.
	servers map[string]*server
}

// serve registers a handler for the given endpoint on a shared HTTP server,
// blocking until the input's context is cancelled or the server dies. Each
// input monitors its own context and deregisters cleanly when stopped; the
// server lives as long as at least one input is registered.
func (p *pool) serve(ctx v2.Context, e *httpEndpoint, pub func(beat.Event), metrics *inputMetrics) error {
	log := ctx.Logger.With("address", e.addr)

	u, err := url.Parse(e.config.URL)
	if err != nil {
		ctx.UpdateStatus(status.Failed, "configured URL is invalid: "+err.Error())
		return err
	}
	pattern := u.Path
	metrics.route.Set(pattern)
	metrics.isTLS.Set(e.tlsConfig != nil)

	var prg *program
	if e.config.Program != "" {
		prg, err = newProgram(e.config.Program, log)
		if err != nil {
			ctx.UpdateStatus(status.Failed, "unable to compile CEL program: "+err.Error())
			return err
		}
	}

	// Derive a per-input handler context from the input's cancellation.
	// This context is cancelled during deregistration so in-flight ACK
	// waits abort before the pipeline client is closed.
	handlerCtx, handlerCancel := context.WithCancel(
		v2.GoContextFromCanceler(ctx.Cancelation),
	)

	p.mu.Lock()
	s, ok := p.servers[e.addr]
	if ok {
		err = checkTLSConsistency(e.addr, s.tls, e.config.TLS)
		if err != nil {
			p.mu.Unlock()
			handlerCancel()
			ctx.UpdateStatus(status.Failed, err.Error())
			return err
		}

		if old, ok := s.idOf[pattern]; ok {
			p.mu.Unlock()
			handlerCancel()
			err = fmt.Errorf("pattern already exists for %s: %s old=%s new=%s",
				e.addr, pattern, old, ctx.ID)
			ctx.UpdateStatus(status.Failed, err.Error())
			return err
		}
		log.Infof("Adding %s end point to server on %s", pattern, e.addr)
		s.mux.add(pattern, newHandler(handlerCtx, e.config, prg, pub, ctx, log, metrics))
		s.idOf[pattern] = ctx.ID
		s.handlerCancel[pattern] = handlerCancel
		p.mu.Unlock()
	} else {
		m := &mux{exact: make(map[string]http.Handler)}
		srv := &http.Server{Addr: e.addr, TLSConfig: e.tlsConfig, Handler: m, ReadHeaderTimeout: 5 * time.Second}
		s = &server{
			idOf:          map[string]string{pattern: ctx.ID},
			handlerCancel: map[string]context.CancelFunc{pattern: handlerCancel},
			tls:           e.config.TLS,
			mux:           m,
			srv:           srv,
			done:          make(chan struct{}),
		}
		s.ctx, s.cancel = context.WithCancel(context.Background())
		m.add(pattern, newHandler(handlerCtx, e.config, prg, pub, ctx, log, metrics))
		p.servers[e.addr] = s
		p.mu.Unlock()

		if e.tlsConfig != nil {
			log.Infof("Starting HTTPS server on %s with %s end point", srv.Addr, pattern)
		} else {
			log.Infof("Starting HTTP server on %s with %s end point", srv.Addr, pattern)
		}
		go func() {
			defaultAddr := ":http"
			if e.tlsConfig != nil {
				defaultAddr = ":https"
			}
			ln, listenErr := listen(s.srv, defaultAddr)
			if listenErr == nil {
				metrics.bindAddr.Set(ln.Addr().String())
				if e.tlsConfig != nil {
					listenErr = s.srv.ServeTLS(ln, "", "")
				} else {
					listenErr = s.srv.Serve(ln)
				}
			}
			s.setErr(listenErr)
			p.mu.Lock()
			delete(p.servers, e.addr)
			p.mu.Unlock()
			s.cancel()
			close(s.done)
		}()
	}

	ctx.UpdateStatus(status.Running, "")

	select {
	case <-ctx.Cancelation.Done():
		ctx.UpdateStatus(status.Stopping, "")
	case <-s.ctx.Done():
		// Server died (listen error or last input on another goroutine
		// closed it). Wait for the listener goroutine to finish so
		// s.err is set before we read it.
		<-s.done
		handlerCancel()
		err := s.getErr()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ctx.UpdateStatus(status.Failed, "server exited unexpectedly: "+err.Error())
		} else {
			ctx.UpdateStatus(status.Stopping, "")
		}
		return err
	}

	// This input was stopped. Deregister under pool lock.
	p.mu.Lock()
	s.handlerCancel[pattern]()
	delete(s.handlerCancel, pattern)
	empty := s.mux.remove(pattern)
	delete(s.idOf, pattern)
	p.mu.Unlock()

	if empty {
		// Tell the listener to stop. Don't delete from the pool here:
		// the listener goroutine removes the pool entry after the port
		// is released, preventing a concurrent creator from getting
		// "address already in use".
		s.srv.Close()
		<-s.done
		return s.getErr()
	}
	return nil
}

func listen(srv *http.Server, defaultAddr string) (net.Listener, error) {
	addr := srv.Addr
	if addr == "" {
		addr = defaultAddr
	}
	return net.Listen("tcp", addr)
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

// server is a collection of HTTP end-points sharing the same underlying
// http.Server. The server's lifetime is independent of any single input:
// it runs until the last input deregisters or the listener returns an error.
type server struct {
	// idOf maps mux pattern to input ID.
	idOf map[string]string
	// handlerCancel maps mux pattern to a function that cancels
	// that handler's context, aborting in-flight ACK waits.
	handlerCancel map[string]context.CancelFunc

	tls *tlscommon.ServerConfig

	mux *mux
	srv *http.Server

	// ctx is cancelled when the server is shutting down (listener
	// returned or last input triggered close). It is independent
	// of any input's context.
	ctx    context.Context
	cancel context.CancelFunc

	// done is closed by the listener goroutine after listenAndServe
	// returns. Waiters use it to ensure s.err is set before reading.
	done chan struct{}

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

// mux is a concurrent-safe HTTP request multiplexer that supports dynamic
// handler registration and removal. It implements the path-matching subset
// of [http.ServeMux] (as of Go 1.21, before the method and wildcard
// extensions in Go 1.22):
//
//   - A pattern without a trailing "/" is an exact match.
//     "/webhooks" matches only the request path "/webhooks".
//
//   - A pattern ending in "/" is a prefix match. The handler receives
//     any request whose path starts with the pattern.
//     "/events/" matches "/events/", "/events/auth0", "/events/a/b", etc.
//
//   - When multiple prefix patterns match a request, the longest one wins.
//     Given "/a/" and "/a/b/", a request for "/a/b/c" matches "/a/b/".
//
//   - An exact match always takes priority over a prefix match.
//     Given "/a/b" (exact) and "/a/" (prefix), a request for "/a/b"
//     matches the exact pattern.
//
//   - If no pattern matches, the request gets a 404.
//
//   - Request paths are cleaned with [path.Clean] before matching.
//     Requests with unclean paths (containing "..", "//", etc.) receive
//     a 301 redirect to the cleaned path, matching [http.ServeMux]
//     behaviour.
//
// Unlike [http.ServeMux], mux does not support host-specific patterns,
// method routing, or Go 1.22 wildcard segments. Handlers can be removed
// at runtime via remove; this is the reason it exists instead of
// using [http.ServeMux] directly.
type mux struct {
	mu     sync.RWMutex
	exact  map[string]http.Handler
	prefix []prefixEntry // sorted longest-first
}

type prefixEntry struct {
	pattern string
	handler http.Handler
}

// add registers a handler for pattern. It is the caller's responsibility
// to check for duplicates before calling add.
func (m *mux) add(pattern string, handler http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.HasSuffix(pattern, "/") {
		m.prefix = append(m.prefix, prefixEntry{pattern: pattern, handler: handler})
		sort.Slice(m.prefix, func(i, j int) bool {
			return len(m.prefix[i].pattern) > len(m.prefix[j].pattern)
		})
	} else {
		m.exact[pattern] = handler
	}
}

// remove deregisters the handler for pattern. It returns true if the mux
// has no remaining handlers.
func (m *mux) remove(pattern string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.HasSuffix(pattern, "/") {
		for i, e := range m.prefix {
			if e.pattern == pattern {
				m.prefix = append(m.prefix[:i], m.prefix[i+1:]...)
				break
			}
		}
	} else {
		delete(m.exact, pattern)
	}
	return len(m.exact) == 0 && len(m.prefix) == 0
}

func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clean := cleanPath(r.URL.Path)
	if clean != r.URL.Path {
		url := *r.URL
		url.Path = clean
		http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
		return
	}
	m.mu.RLock()
	h := m.match(clean)
	m.mu.RUnlock()
	if h == nil {
		http.NotFound(w, r)
		return
	}
	h.ServeHTTP(w, r)
}

// cleanPath returns the canonical path for p, eliminating . and .. elements.
// A trailing slash is preserved, matching [http.ServeMux] behaviour.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// match returns the best handler for path. Caller must hold at least a
// read lock on m.mu.
func (m *mux) match(path string) http.Handler {
	if h, ok := m.exact[path]; ok {
		return h
	}
	for _, e := range m.prefix {
		if strings.HasPrefix(path, e.pattern) {
			return e.handler
		}
	}
	return nil
}

func newHandler(ctx context.Context, c config, prg *program, pub func(beat.Event), stat status.StatusReporter, log *logp.Logger, metrics *inputMetrics) http.Handler {
	h := &handler{
		ctx:      ctx,
		log:      log,
		txBaseID: newID(),

		status:  stat,
		publish: pub,
		metrics: metrics,
		validator: apiValidator{
			basicAuth:      c.BasicAuth,
			username:       c.Username,
			password:       c.Password,
			method:         c.Method,
			contentType:    c.ContentType,
			secretHeader:   c.SecretHeader,
			secretValue:    c.SecretValue,
			hmacHeader:     c.HMACHeader,
			hmacKey:        c.HMACKey,
			hmacType:       c.HMACType,
			hmacPrefix:     c.HMACPrefix,
			maxBodySize:    -1,
			optionsHeaders: c.OptionsHeaders,
			optionsStatus:  c.OptionsStatus,
		},
		maxInFlight:           c.MaxInFlight,
		highWaterInFlight:     c.HighWaterInFlight,
		lowWaterInFlight:      c.LowWaterInFlight,
		retryAfter:            c.RetryAfter,
		program:               prg,
		messageField:          c.Prefix,
		responseCode:          c.ResponseCode,
		responseBody:          htmlEscape(c.ResponseBody),
		includeHeaders:        canonicalizeHeaders(c.IncludeHeaders),
		preserveOriginalEvent: c.PreserveOriginalEvent,
		crc:                   newCRC(c.CRCProvider, c.CRCSecret),
	}
	// Initialize accepting to true so we start by accepting requests.
	h.accepting.Store(true)
	if h.status == nil {
		h.status = noopReporter{}
	}
	if c.MaxBodySize != nil {
		h.validator.maxBodySize = *c.MaxBodySize
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
		h.host = net.JoinHostPort(c.ListenAddress, c.ListenPort)
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

type noopReporter struct{}

func (noopReporter) UpdateStatus(status.Status, string) {}

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

func newInputMetrics(reg *monitoring.Registry, logger *logp.Logger) *inputMetrics {
	out := &inputMetrics{
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
	_ = adapter.NewGoMetrics(reg, "size", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.contentLength))
	_ = adapter.NewGoMetrics(reg, "batch_size", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchSize))
	_ = adapter.NewGoMetrics(reg, "batch_processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime))
	_ = adapter.NewGoMetrics(reg, "batch_ack_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchACKTime))

	return out
}
