// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/elastic/beats/libbeat/logp"
)

// cfClient interface is provided so unit tests can mock the actual client.
type cfClient interface {
	// GetAppByGuid returns an application information from its Guid.
	GetAppByGuid(guid string) (cfclient.App, error)
}

// clientCacheWrap wraps the cloudfoundry client to add a cache in front of GetAppByGuid.
type clientCacheWrap struct {
	store  cache.Store
	client cfClient
	log    *logp.Logger
	ttl    time.Duration
}

// newClientCacheWrap creates a new cache for application data.
func newClientCacheWrap(client cfClient, ttl time.Duration, log *logp.Logger) *clientCacheWrap {
	return &clientCacheWrap{
		store:  cache.NewTTLStore(cacheKeyFunc, ttl),
		client: client,
		ttl:    ttl,
		log:    log,
	}
}

// appCached wraps an App structure adding a retrieval time
type appCached struct {
	app cfclient.App
	ttl time.Time
}

// fetchApp uses the cfClient to retrieve an App entity and
// stores it in the internal cache
func (c *clientCacheWrap) fetchAppByGuid(guid string) (*cfclient.App, error) {
	app, err := c.client.GetAppByGuid(guid)
	if err != nil {
		return nil, err
	}
	ttl := time.Now().Add(c.ttl)
	c.store.Add(&appCached{app, ttl})
	return &app, nil
}

// GetApp returns CF Application info, either from the cache or
// using the CF client.
func (c *clientCacheWrap) GetAppByGuid(guid string) (*cfclient.App, error) {
	cachedApp, ok, _ := c.store.GetByKey(guid)
	if !ok {
		return c.fetchAppByGuid(guid)
	}

	ac, ok := cachedApp.(*appCached)
	if !ok {
		return nil, fmt.Errorf("error converting cached app")
	}

	if ac.ttl.Before(time.Now()) {
		c.log.Debugf("cached data for application %q invalidated, retrieving again", guid)
		return c.fetchAppByGuid(guid)
	}

	return &ac.app, nil
}

func cacheKeyFunc(obj interface{}) (string, error) {
	cachedApp, ok := obj.(*appCached)
	if !ok {
		return "", fmt.Errorf("only appCached is allowed in the cache")
	}
	return cachedApp.app.Guid, nil
}
