// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// cfClient interface is provided so unit tests can mock the actual client.
type cfClient interface {
	// GetAppByGuid returns an application information from its Guid.
	GetAppByGuid(guid string) (cfclient.App, error)
}

// internalCache is the interface for internal caches
type internalCache interface {
	PutWithTimeout(common.Key, common.Value, time.Duration) common.Value
	Get(common.Key) common.Value
	StartJanitor(time.Duration)
	StopJanitor()
}

// openCloser is the interface for resources that need to be opened and closed
// before and after using them.
type openCloser interface {
	Open() error
	Close() error
}

// clientCacheWrap wraps the cloudfoundry client to add a cache in front of GetAppByGuid.
type clientCacheWrap struct {
	cache    internalCache
	client   cfClient
	log      *logp.Logger
	errorTTL time.Duration
}

// newClientCacheWrap creates a new cache for application data.
func newClientCacheWrap(client cfClient, ttl time.Duration, errorTTL time.Duration, log *logp.Logger) *clientCacheWrap {
	return &clientCacheWrap{
		cache:    common.NewCacheWithExpireOnAdd(ttl, 100),
		client:   client,
		errorTTL: errorTTL,
		log:      log,
	}
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
	cachedResp := c.cache.Get(guid)
	if cachedResp == nil {
		return c.fetchAppByGuid(guid)
	}
	resp, ok := cachedResp.(*appResponse)
	if !ok {
		return nil, fmt.Errorf("error converting cached app response (of type %T), this is likely a bug", cachedResp)
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
