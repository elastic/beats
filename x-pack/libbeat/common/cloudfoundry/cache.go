// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/persistentcache"
)

// cfClient interface is provided so unit tests can mock the actual client.
type cfClient interface {
	// GetAppByGuid returns an application information from its Guid.
	GetAppByGuid(guid string) (cfclient.App, error)
}

// clientCacheWrap wraps the cloudfoundry client to add a cache in front of GetAppByGuid.
type clientCacheWrap struct {
	cache    *persistentcache.PersistentCache
	client   cfClient
	log      *logp.Logger
	errorTTL time.Duration
}

// newClientCacheWrap creates a new cache for application data.
func newClientCacheWrap(client cfClient, ttl time.Duration, errorTTL time.Duration, log *logp.Logger) (*clientCacheWrap, error) {
	options := persistentcache.Options{
		Timeout: ttl,
	}

	// TODO: Use an unique name per API endpoint
	cache, err := persistentcache.New("cloudfoundry", options)
	if err != nil {
		return nil, fmt.Errorf("creating metadata cache: %w", err)
	}

	return &clientCacheWrap{
		cache:    cache,
		client:   client,
		errorTTL: errorTTL,
		log:      log,
	}, nil
}

type appResponse struct {
	// TODO: Store only the fields we need from the App.
	App          cfclient.App               `json:"a"`
	Error        cfclient.CloudFoundryError `json:"e,omitempty"`
	ErrorMessage string                     `json:"em,omitempty"`
}

func (r *appResponse) fromStructs(app cfclient.App, err error) {
	if err != nil {
		switch e := err.(type) {
		case cfclient.CloudFoundryError:
			// Store native CF errors as they are. They are serializable and
			// contain relevant information.
			r.Error = e
		default:
			r.ErrorMessage = e.Error()
		}
		return
	}
	r.App = app
}

func (r *appResponse) toStructs() (*cfclient.App, error) {
	if len(r.ErrorMessage) > 0 {
		return nil, errors.New(r.ErrorMessage)
	}
	var empty cfclient.CloudFoundryError
	if r.Error != empty {
		return nil, r.Error
	}
	return &r.App, nil
}

// fetchApp uses the cfClient to retrieve an App entity and
// stores it in the internal cache
func (c *clientCacheWrap) fetchAppByGuid(guid string) (*cfclient.App, error) {
	app, err := c.client.GetAppByGuid(guid)
	var resp appResponse
	resp.fromStructs(app, err)
	timeout := time.Duration(0)
	if err != nil {
		// Cache nil, because is what we want to return when there was an error
		timeout = c.errorTTL
	}
	err = c.cache.PutWithTimeout(guid, resp, timeout)
	if err != nil {
		return nil, fmt.Errorf("storing app response in cache: %w", err)
	}
	return resp.toStructs()
}

// GetApp returns CF Application info, either from the cache or
// using the CF client.
func (c *clientCacheWrap) GetAppByGuid(guid string) (*cfclient.App, error) {
	var resp appResponse
	err := c.cache.Get(guid, &resp)
	if err != nil {
		return c.fetchAppByGuid(guid)
	}
	return resp.toStructs()
}

// StartJanitor starts a goroutine that will periodically clean the applications cache.
func (c *clientCacheWrap) StartJanitor(interval time.Duration) {
	c.cache.StartJanitor(interval)
}

// StopJanitor stops the goroutine that periodically clean the applications cache.
func (c *clientCacheWrap) StopJanitor() {
	c.cache.StopJanitor()
}
