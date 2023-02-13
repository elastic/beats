// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
		Stability:  feature.Beta,
		Deprecated: false,
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
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

func (e *httpEndpoint) Run(ctx v2.Context, publisher stateless.Publisher) error {
	err := servers.serve(ctx, e, publisher)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("unable to start server due to error: %w", err)
	}
	return nil
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
func (p *pool) serve(ctx v2.Context, e *httpEndpoint, pub stateless.Publisher) error {
	log := ctx.Logger.With("address", e.addr)
	pattern := e.config.URL

	var err error
	p.mu.Lock()
	s, ok := p.servers[e.addr]
	if ok {
		err = checkTLSConsistency(e.addr, s.tls, e.config.TLS)
		if err != nil {
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
		s.mux.Handle(pattern, newHandler(e.config, pub, log))
		s.idOf[pattern] = ctx.ID
		p.mu.Unlock()
		<-s.ctx.Done()
		return s.getErr()
	}

	mux := http.NewServeMux()
	mux.Handle(pattern, newHandler(e.config, pub, log))
	srv := &http.Server{Addr: e.addr, TLSConfig: e.tlsConfig, Handler: mux}
	s = &server{
		idOf: map[string]string{pattern: ctx.ID},
		tls:  e.config.TLS,
		mux:  mux,
		srv:  srv,
	}
	s.ctx, s.cancel = ctxtool.WithFunc(ctx.Cancelation, func() { srv.Close() })
	p.servers[e.addr] = s
	p.mu.Unlock()

	if e.tlsConfig != nil {
		log.Infof("Starting HTTPS server on %s with %s end point", srv.Addr, pattern)
		// The certificate is already loaded so we do not need
		// to pass the cert file and key file parameters.
		err = s.srv.ListenAndServeTLS("", "")
	} else {
		log.Infof("Starting HTTP server on %s with %s end point", srv.Addr, pattern)
		err = s.srv.ListenAndServe()
	}
	s.setErr(err)
	s.cancel()
	return err
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

func newHandler(c config, pub stateless.Publisher, log *logp.Logger) http.Handler {
	validator := &apiValidator{
		basicAuth:    c.BasicAuth,
		username:     c.Username,
		password:     c.Password,
		method:       http.MethodPost,
		contentType:  c.ContentType,
		secretHeader: c.SecretHeader,
		secretValue:  c.SecretValue,
		hmacHeader:   c.HMACHeader,
		hmacKey:      c.HMACKey,
		hmacType:     c.HMACType,
		hmacPrefix:   c.HMACPrefix,
	}

	handler := &httpHandler{
		log:                   log,
		publisher:             pub,
		messageField:          c.Prefix,
		responseCode:          c.ResponseCode,
		responseBody:          c.ResponseBody,
		includeHeaders:        canonicalizeHeaders(c.IncludeHeaders),
		preserveOriginalEvent: c.PreserveOriginalEvent,
	}

	return newAPIValidationHandler(http.HandlerFunc(handler.apiResponse), validator, log)
}
