// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
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
	app *cfclient.App
	err error
}

// fetchApp uses the cfClient to retrieve an App entity and
// stores it in the internal cache
func (c *clientCacheWrap) fetchAppByGuid(guid string) (*cfclient.App, error) {
	app, err := c.client.GetAppByGuid(guid)
	resp := appResponse{
		app: &app,
		err: err,
	}
	timeout := time.Duration(0)
	if err != nil {
		// Cache nil, because is what we want to return when there was an error
		resp.app = nil
		timeout = c.errorTTL
	}
	c.cache.PutWithTimeout(guid, &resp, timeout)
	return resp.app, resp.err
}

// GetApp returns CF Application info, either from the cache or
// using the CF client.
func (c *clientCacheWrap) GetAppByGuid(guid string) (*cfclient.App, error) {
	var resp appResponse
	err := c.cache.Get(guid, &resp)
	if err != nil {
		return c.fetchAppByGuid(guid)
	}
	return resp.app, resp.err
}

// StartJanitor starts a goroutine that will periodically clean the applications cache.
func (c *clientCacheWrap) StartJanitor(interval time.Duration) {
	c.cache.StartJanitor(interval)
}

// StopJanitor stops the goroutine that periodically clean the applications cache.
func (c *clientCacheWrap) StopJanitor() {
	c.cache.StopJanitor()
}
