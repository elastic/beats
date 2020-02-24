// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-incubator/uaago"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
)

// Client interface exposed by Hub.Client.
type Client interface {
	GetAppByGuid(guid string) (*cfclient.App, error)
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
func (h *Hub) Client() (Client, error) {
	h.log.Debugw(
		"creating cloudfoundry ",
		"client_id", h.cfg.ClientID,
		"client_secret_present", h.cfg.ClientSecret != "",
		"skip_validation", h.cfg.SkipVerify,
		"api_address", h.cfg.APIAddress)
	cf, err := cfclient.NewClient(&cfclient.Config{
		ClientID:          h.cfg.ClientID,
		ClientSecret:      h.cfg.ClientSecret,
		ApiAddress:        h.cfg.APIAddress,
		SkipSslValidation: h.cfg.SkipVerify,
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
	return newClientCacheWrap(cf, h.cfg.CacheDuration, h.log), nil
}

// Uaa returns the uaa cloudfoundry client.
func (h *Hub) Uaa() (*uaago.Client, error) {
	client, err := h.Client()
	if err != nil {
		return nil, err
	}
	return h.UaaFromClient(client)
}

// UaaFromClient returns the uaa cloudfoundry client from the provided client.
//
// In the case that the cloudfoundry client was already needed by the code path, call this method
// as not to create a intermediate client that will not be used.
func (h *Hub) UaaFromClient(client Client) (*uaago.Client, error) {
	wrapper, ok := client.(*clientCacheWrap)
	if !ok {
		return nil, fmt.Errorf("must pass in a client returned from Hub.Client()")
	}
	cfClient, ok := wrapper.client.(*cfclient.Client)
	if !ok {
		return nil, fmt.Errorf("client.client is not a cfclient.Client")
	}
	h.log.Debugw("creating UAA client", "AuthEndpoint", cfClient.Endpoint.AuthEndpoint)
	return uaago.NewClient(cfClient.Endpoint.AuthEndpoint)
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
func (h *Hub) RlpListenerFromClient(client Client, callbacks RlpListenerCallbacks) (*RlpListener, error) {
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

// doerFromClient returns an auth token doer using uaa.
func (h *Hub) doerFromClient(client Client) (*authTokenDoer, error) {
	uaa, err := h.UaaFromClient(client)
	if err != nil {
		return nil, err
	}
	return newAuthTokenDoer(uaa, h.cfg.ClientID, h.cfg.ClientSecret, h.cfg.SkipVerify, h.log), nil
}
