// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	defaultPort = 8220

	retryOnBadConnTimeout = 5 * time.Minute
)

var hasScheme = regexp.MustCompile(`^([a-z][a-z0-9+\-.]*)://`)

type requestFunc func(string, string, url.Values, io.Reader) (*http.Request, error)
type wrapperFunc func(rt http.RoundTripper) (http.RoundTripper, error)

type requestClient struct {
	request    requestFunc
	client     http.Client
	lastUsed   time.Time
	lastErr    error
	lastErrOcc time.Time
}

// Client wraps an http.Client and takes care of making the raw calls, the client should
// stay simple and specificals should be implemented in external action instead of adding new methods
// to the client. For authenticated calls or sending fields on every request, create customer RoundTripper
// implementations that will take care of the boiler plates.
type Client struct {
	log     *logger.Logger
	lock    sync.Mutex
	clients []*requestClient
	config  Config
}

// NewConfigFromURL returns a Config based on a received host.
func NewConfigFromURL(kURL string) (Config, error) {
	u, err := url.Parse(kURL)
	if err != nil {
		return Config{}, errors.Wrap(err, "could not parse url")
	}

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		// _ is true when password is set.
		password, _ = u.User.Password()
	}

	c := DefaultClientConfig()
	c.Protocol = Protocol(u.Scheme)
	c.Host = u.Host
	c.Path = u.Path
	c.Username = username
	c.Password = password

	return c, nil
}

// NewWithRawConfig returns a new client with a specified configuration.
func NewWithRawConfig(log *logger.Logger, config *config.Config, wrapper wrapperFunc) (*Client, error) {
	l := log
	if l == nil {
		log, err := logger.New("client", false)
		if err != nil {
			return nil, err
		}
		l = log
	}

	cfg := Config{}
	if err := config.Unpack(&cfg); err != nil {
		return nil, errors.Wrap(err, "invalidate configuration")
	}

	return NewWithConfig(l, cfg, wrapper)
}

// NewWithConfig takes a Config and return a client.
func NewWithConfig(log *logger.Logger, cfg Config, wrapper wrapperFunc) (*Client, error) {
	// Normalize the URL with the path any spaces configured.
	var p string
	if len(cfg.SpaceID) > 0 {
		p = strings.Join([]string{cfg.Path, cfg.SpaceID}, "/")
	} else {
		p = cfg.Path
	}

	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}

	hosts := cfg.GetHosts()
	clients := make([]*requestClient, len(hosts))
	for i, host := range cfg.GetHosts() {
		connStr, err := common.MakeURL(string(cfg.Protocol), p, host, 0)
		if err != nil {
			return nil, errors.Wrap(err, "invalid fleet-server endpoint")
		}

		transport, err := cfg.Transport.RoundTripper(
			httpcommon.WithAPMHTTPInstrumentation(),
			httpcommon.WithForceAttemptHTTP2(true),
		)
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
			Timeout:   cfg.Transport.Timeout,
		}

		clients[i] = &requestClient{
			request: prefixRequestFactory(connStr),
			client:  httpClient,
		}
	}

	return new(log, cfg, clients...)
}

// Send executes a direct calls against the API, the method will takes cares of cloning
// also add necessary headers for likes: "Content-Type", "Accept", and "kbn-xsrf".
// No assumptions is done on the response concerning the received format, this will be the responsibility
// of the implementation to correctly unpack any received data.
//
// NOTE:
// - The caller of this method is free to override any value found in the headers.
// - The magic of unpacking of errors is not done in the Send method, a helper method is provided.
func (c *Client) Send(
	ctx context.Context,
	method, path string,
	params url.Values,
	headers http.Header,
	body io.Reader,
) (*http.Response, error) {
	c.log.Debugf("Request method: %s, path: %s", method, path)
	c.lock.Lock()
	defer c.lock.Unlock()
	requester := c.nextRequester()

	req, err := requester.request(method, path, params, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create HTTP request using method %s to %s", method, path)
	}
	c.log.Debugf("Creating new request to request URL %s", req.URL.String())

	// Add generals headers to the request, we are dealing exclusively with JSON.
	// Content-Type / Accepted type can be override from the called.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	// TODO: Make this header specific to fleet-server or remove it
	req.Header.Set("kbn-xsrf", "1") // Without this Kibana will refuse to answer the request.

	// copy headers.
	for header, values := range headers {
		for _, v := range values {
			req.Header.Add(header, v)
		}
	}

	requester.lastUsed = time.Now().UTC()

	resp, err := requester.client.Do(req.WithContext(ctx))
	if err != nil {
		requester.lastErr = err
		requester.lastErrOcc = time.Now().UTC()
	} else {
		requester.lastErr = nil
		requester.lastErrOcc = time.Time{}
	}
	return resp, err
}

// URI returns the remote URI.
func (c *Client) URI() string {
	host := c.config.GetHosts()[0]
	return string(c.config.Protocol) + "://" + host + "/" + c.config.Path
}

// new creates new API client.
func new(
	log *logger.Logger,
	cfg Config,
	httpClients ...*requestClient,
) (*Client, error) {
	c := &Client{
		log:     log,
		clients: httpClients,
		config:  cfg,
	}
	return c, nil
}

// nextRequester returns the requester to use.
//
// It excludes clients that have errored in the last 5 minutes.
func (c *Client) nextRequester() *requestClient {
	var selected *requestClient

	now := time.Now().UTC()
	for _, requester := range c.clients {
		if requester.lastErr != nil && now.Sub(requester.lastErrOcc) > retryOnBadConnTimeout {
			requester.lastErr = nil
			requester.lastErrOcc = time.Time{}
		}
		if requester.lastErr != nil {
			continue
		}
		if requester.lastUsed.IsZero() {
			// never been used, instant winner!
			selected = requester
			break
		}
		if selected == nil {
			selected = requester
			continue
		}
		if requester.lastUsed.Before(selected.lastUsed) {
			selected = requester
		}
	}
	if selected == nil {
		// all are erroring; select the oldest one that errored
		for _, requester := range c.clients {
			if selected == nil {
				selected = requester
				continue
			}
			if requester.lastErrOcc.Before(selected.lastErrOcc) {
				selected = requester
			}
		}
	}
	return selected
}

func prefixRequestFactory(URL string) requestFunc {
	return func(method, path string, params url.Values, body io.Reader) (*http.Request, error) {
		path = strings.TrimPrefix(path, "/")
		newPath := strings.Join([]string{URL, path, "?", params.Encode()}, "")
		return http.NewRequest(method, newPath, body)
	}
}
