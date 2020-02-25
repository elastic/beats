// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

const kibanaPort = 5601

type requestFunc func(string, string, url.Values, io.Reader) (*http.Request, error)
type wrapperFunc func(rt http.RoundTripper) (http.RoundTripper, error)

type clienter interface {
	Send(
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)
	Close() error
}

// Client wraps an http.Client and takes care of making the raw calls to kibana, the client should
// stay simple and specificals should be implemented in external action instead of adding new methods
// to the client. For authenticated calls or sending fields on every request, create customer RoundTripper
// implementations that will take care of the boiler plates.
type Client struct {
	log     *logger.Logger
	request requestFunc
	client  http.Client
	config  *Config
}

// New creates new Kibana API client.
func New(
	log *logger.Logger,
	factory requestFunc,
	cfg *Config,
	httpClient http.Client,
) (*Client, error) {
	c := &Client{
		log:     log,
		request: factory,
		client:  httpClient,
		config:  cfg,
	}
	return c, nil
}

// NewConfigFromURL returns a Kibana Config based on a received host.
func NewConfigFromURL(kURL string) (*Config, error) {
	u, err := url.Parse(kURL)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse Kibana url")
	}

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		// _ is true when password is set.
		password, _ = u.User.Password()
	}

	c := defaultClientConfig()
	c.Protocol = Protocol(u.Scheme)
	c.Host = u.Host
	c.Path = u.Path
	c.Username = username
	c.Password = password

	return &c, nil
}

// NewWithRawConfig returns a new Kibana client with a specified configuration.
func NewWithRawConfig(log *logger.Logger, config *config.Config, wrapper wrapperFunc) (*Client, error) {
	l := log
	if l == nil {
		log, err := logger.New()
		if err != nil {
			return nil, err
		}
		l = log
	}

	cfg := &Config{}
	if err := config.Unpack(cfg); err != nil {
		return nil, errors.Wrap(err, "invidate configuration")
	}

	return NewWithConfig(l, cfg, wrapper)
}

// NewWithConfig takes a Kibana Config and return a client.
func NewWithConfig(log *logger.Logger, cfg *Config, wrapper wrapperFunc) (*Client, error) {
	var transport http.RoundTripper
	transport, err := makeTransport(cfg.Timeout, cfg.TLS)
	if err != nil {
		return nil, err
	}

	if cfg.IsBasicAuth() {
		// Pass basic auth credentials to all the underlying calls.
		transport = NewBasicAuthRoundTripper(transport, cfg.Username, cfg.Password)
	}

	if wrapper != nil {
		transport, err = wrapper(transport)
		if err != nil {
			return nil, errors.Wrap(err, "fail to create transport client")
		}
	}

	httpClient := http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	p := strings.Join([]string{cfg.Path, cfg.SpaceID}, "/")

	kibanaURL, err := common.MakeURL(string(cfg.Protocol), p, cfg.Host, kibanaPort)
	if err != nil {
		return nil, errors.Wrap(err, "invalid Kibana endpoint")
	}

	return New(log, prefixRequestFactory(kibanaURL), cfg, httpClient)
}

// Send executes a direct calls agains't the Kibana API, the method will takes cares of cloning
// also add necessary headers for Kibana likes: "Content-Type", "Accept", and "kbn-xsrf".
// No assumptions is done on the response concerning the received format, this will be the responsability
// of the implementation to correctly unpack any received data.
//
// NOTE:
// - The caller of this method is free to overrides any values found in the headers.
// - The magic of unpack kibana errors is not done in the Send method, an helper methods is provided.
func (c *Client) Send(
	ctx context.Context,
	method, path string,
	params url.Values,
	headers http.Header,
	body io.Reader,
) (*http.Response, error) {
	c.log.Debugf("Request method: %s, path: %s", method, path)

	req, err := c.request(method, path, params, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create HTTP request using method %s to %s", method, path)
	}

	// Add generals headers to the request, we are dealing exclusively with JSON.
	// Content-Type / Accepted type can be override from the called.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Set("kbn-xsrf", "1") // Without this Kibana will refuse to answer the request.

	// copy headers.
	for header, values := range headers {
		for _, v := range values {
			req.Header.Add(header, v)
		}
	}

	return c.client.Do(req.WithContext(ctx))
}

// URI returns the remote URI.
func (c *Client) URI() string {
	return string(c.config.Protocol) + "://" + c.config.Host + "/" + c.config.Path
}

func prefixRequestFactory(URL string) requestFunc {
	return func(method, path string, params url.Values, body io.Reader) (*http.Request, error) {
		path = strings.TrimPrefix(path, "/")
		newPath := strings.Join([]string{URL, path, "?", params.Encode()}, "")
		return http.NewRequest(method, newPath, body)
	}
}

// makeTransport create a transport object based on the TLS configuration.
func makeTransport(timeout time.Duration, tls *tlscommon.Config) (*http.Transport, error) {
	tlsConfig, err := tlscommon.LoadTLSConfig(tls)
	if err != nil {
		return nil, errors.Wrap(err, "invalid TLS configuration")
	}
	dialer := transport.NetDialer(timeout)
	tlsDialer, err := transport.TLSDialer(dialer, tlsConfig, timeout)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create TLS dialer")
	}

	// TODO: Dial is deprecated we need to move to DialContext.
	return &http.Transport{Dial: dialer.Dial, DialTLS: tlsDialer.Dial}, nil
}
