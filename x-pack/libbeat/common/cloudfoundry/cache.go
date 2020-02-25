// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// cfClient interface is provided so unit tests can mock the actual client.
type cfClient interface {
	// GetAppByGuid returns an application information from its Guid.
	GetAppByGuid(guid string) (cfclient.App, error)
}

// clientCacheWrap wraps the cloudfoundry client to add a cache in front of GetAppByGuid.
type clientCacheWrap struct {
	cache  *common.Cache
	client cfClient
	log    *logp.Logger
}

// newClientCacheWrap creates a new cache for application data.
func newClientCacheWrap(client cfClient, ttl time.Duration, log *logp.Logger) *clientCacheWrap {
	return &clientCacheWrap{
		cache:  common.NewCacheWithExpireOnAdd(ttl, 100),
		client: client,
		log:    log,
	}
}

// fetchApp uses the cfClient to retrieve an App entity and
// stores it in the internal cache
func (c *clientCacheWrap) fetchAppByGuid(guid string) (*cfclient.App, error) {
	app, err := c.client.GetAppByGuid(guid)
	if err != nil {
		return nil, err
	}
	c.cache.Put(app.Guid, &app)
	return &app, nil
}

// GetApp returns CF Application info, either from the cache or
// using the CF client.
func (c *clientCacheWrap) GetAppByGuid(guid string) (*cfclient.App, error) {
	cachedApp := c.cache.Get(guid)
	if cachedApp == nil {
		return c.fetchAppByGuid(guid)
	}
	app, ok := cachedApp.(*cfclient.App)
	if !ok {
		return nil, fmt.Errorf("error converting cached app")
	}
	return app, nil
}

// StartJanitor starts a goroutine that will periodically clean the applications cache.
func (c *clientCacheWrap) StartJanitor(interval time.Duration) {
	c.cache.StartJanitor(interval)
}

// StopJanitor stops the goroutine that periodically clean the applications cache.
func (c *clientCacheWrap) StopJanitor() {
	c.cache.StopJanitor()
}
