// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/feature"
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

func configure(cfg *common.Config) (stateless.Input, error) {
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
		tlsConfig = tlsConfigBuilder.BuildModuleClientConfig(addr)
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
	log := ctx.Logger.With("address", e.addr)

	validator := &apiValidator{
		basicAuth:    e.config.BasicAuth,
		username:     e.config.Username,
		password:     e.config.Password,
		method:       http.MethodPost,
		contentType:  e.config.ContentType,
		secretHeader: e.config.SecretHeader,
		secretValue:  e.config.SecretValue,
		hmacHeader:   e.config.HMACHeader,
		hmacKey:      e.config.HMACKey,
		hmacType:     e.config.HMACType,
		hmacPrefix:   e.config.HMACPrefix,
	}

	handler := &httpHandler{
		log:                   log,
		publisher:             publisher,
		messageField:          e.config.Prefix,
		responseCode:          e.config.ResponseCode,
		responseBody:          e.config.ResponseBody,
		includeHeaders:        canonicalizeHeaders(e.config.IncludeHeaders),
		preserveOriginalEvent: e.config.PreserveOriginalEvent,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(e.config.URL, withValidator(validator, handler.apiResponse))
	server := &http.Server{Addr: e.addr, TLSConfig: e.tlsConfig, Handler: mux}
	_, cancel := ctxtool.WithFunc(ctx.Cancelation, func() { server.Close() })
	defer cancel()

	var err error
	if server.TLSConfig != nil {
		log.Infof("Starting HTTPS server on %s", server.Addr)
		// certificate is already loaded. That's why the parameters are empty
		err = server.ListenAndServeTLS("", "")
	} else {
		log.Infof("Starting HTTP server on %s", server.Addr)
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("unable to start server due to error: %w", err)
	}
	return nil
}
