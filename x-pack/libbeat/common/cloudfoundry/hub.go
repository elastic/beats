// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"net/http"
	"strings"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Client interface exposed by Hub.Client.
type Client interface {
	// GetAppByGuid returns the application from cloudfoundry.
	GetAppByGuid(guid string) (*AppMeta, error)

	// Close releases resources associated with this client.
	Close() error
}

// AppMeta is the metadata associated with a cloudfoundry application
type AppMeta struct {
	Guid      string `json:"guid"`
	Name      string `json:"name"`
	SpaceGuid string `json:"space_guid"`
	SpaceName string `json:"space_name"`
	OrgGuid   string `json:"org_guid"`
	OrgName   string `json:"org_name"`
}

// Hub is central place to get all the required clients to communicate with cloudfoundry.
type Hub struct {
	cfg       *Config
	userAgent string
	log       *logp.Logger
}

// NewHub creates a new hub to get the required clients to communicate with cloudfoundry.
func NewHub(cfg *Config, userAgent string, log *logp.Logger) *Hub {
	return &Hub{cfg, userAgent, log}
}

// Client returns the cloudfoundry client.
func (h *Hub) Client() (*cfclient.Client, error) {
	httpClient, insecure, err := h.httpClient()
	if err != nil {
		return nil, err
	}

	h.log.Debugw(
		"creating cloudfoundry ",
		"client_id", h.cfg.ClientID,
		"client_secret_present", h.cfg.ClientSecret != "",
		"api_address", h.cfg.APIAddress)
	cf, err := cfclient.NewClient(&cfclient.Config{
		ClientID:          h.cfg.ClientID,
		ClientSecret:      h.cfg.ClientSecret,
		ApiAddress:        h.cfg.APIAddress,
		HttpClient:        httpClient,
		SkipSslValidation: insecure,
		UserAgent:         h.userAgent,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating cloudfoundry client")
	}
	if h.cfg.DopplerAddress != "" {
		cf.Endpoint.DopplerEndpoint = h.cfg.DopplerAddress
	}
	if h.cfg.UaaAddress != "" {
		cf.Endpoint.AuthEndpoint = h.cfg.UaaAddress
	}
	return cf, nil
}

func (h *Hub) ClientWithCache() (Client, error) {
	c, err := h.Client()
	if err != nil {
		return nil, err
	}
	return newClientCacheWrap(c, h.cfg.APIAddress, h.cfg.CacheDuration, h.cfg.CacheRetryDelay, h.log)
}

// RlpListener returns a listener client that calls the passed callback when the provided events are streamed through
// the loggregator to this client.
func (h *Hub) RlpListener(callbacks RlpListenerCallbacks) (*RlpListener, error) {
	client, err := h.Client()
	if err != nil {
		return nil, err
	}
	return h.RlpListenerFromClient(client, callbacks)
}

// RlpListener returns a listener client that calls the passed callback when the provided events are streamed through
// the loggregator to this client.
//
// In the case that the cloudfoundry client was already needed by the code path, call this method
// as not to create a intermediate client that will not be used.
func (h *Hub) RlpListenerFromClient(client *cfclient.Client, callbacks RlpListenerCallbacks) (*RlpListener, error) {
	var rlpAddress string
	if h.cfg.RlpAddress != "" {
		rlpAddress = h.cfg.RlpAddress
	} else {
		rlpAddress = strings.Replace(h.cfg.APIAddress, "api", "log-stream", 1)
	}
	doer, err := h.doerFromClient(client)
	if err != nil {
		return nil, err
	}
	return newRlpListener(rlpAddress, doer, h.cfg.ShardID, callbacks, h.log), nil
}

func (h *Hub) DopplerConsumer(callbacks DopplerCallbacks) (*DopplerConsumer, error) {
	client, err := h.Client()
	if err != nil {
		return nil, err
	}
	return h.DopplerConsumerFromClient(client, callbacks)
}

func (h *Hub) DopplerConsumerFromClient(client *cfclient.Client, callbacks DopplerCallbacks) (*DopplerConsumer, error) {
	dopplerAddress := h.cfg.DopplerAddress
	if dopplerAddress == "" {
		dopplerAddress = client.Endpoint.DopplerEndpoint
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(h.cfg.Transport.TLS)
	if err != nil {
		return nil, errors.Wrap(err, "loading tls config")
	}
	proxy := h.cfg.Transport.Proxy.ProxyFunc()

	tr := TokenRefresherFromCfClient(client)
	return newDopplerConsumer(dopplerAddress, h.cfg.ShardID, h.log, tlsConfig.ToConfig(), proxy, tr, callbacks)
}

// doerFromClient returns an auth token doer using uaa.
func (h *Hub) doerFromClient(client *cfclient.Client) (*authTokenDoer, error) {
	httpClient, _, err := h.httpClient()
	if err != nil {
		return nil, err
	}
	url := h.cfg.UaaAddress
	if url == "" {
		url = client.Endpoint.AuthEndpoint
	}
	return newAuthTokenDoer(url, h.cfg.ClientID, h.cfg.ClientSecret, httpClient, h.log), nil
}

// httpClient returns an HTTP client configured with the configuration TLS.
func (h *Hub) httpClient() (*http.Client, bool, error) {
	httpClient, err := h.cfg.Transport.Client(httpcommon.WithAPMHTTPInstrumentation())
	if err != nil {
		return nil, false, err
	}

	tls, _ := tlscommon.LoadTLSConfig(h.cfg.Transport.TLS)
	return httpClient, tls.ToConfig().InsecureSkipVerify, nil
}
