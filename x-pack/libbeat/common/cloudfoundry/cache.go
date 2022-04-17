// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/libbeat/persistentcache"
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
func newClientCacheWrap(client cfClient, cacheName string, ttl time.Duration, errorTTL time.Duration, log *logp.Logger) (*clientCacheWrap, error) {
	options := persistentcache.Options{
		Timeout: ttl,
	}

	name := "cloudfoundry"
	if cacheName != "" {
		name = name + "-" + sanitizeCacheName(cacheName)
	}

	cache, err := persistentcache.New(name, options)
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
	App          AppMeta                    `json:"a"`
	Error        cfclient.CloudFoundryError `json:"e,omitempty"`
	ErrorMessage string                     `json:"em,omitempty"`
}

func (r *appResponse) fromStructs(app cfclient.App, err error) {
	if err != nil {
		cause := errors.Cause(err)
		if cferr, ok := cause.(cfclient.CloudFoundryError); ok {
			r.Error = cferr
		}
		r.ErrorMessage = err.Error()
		return
	}
	r.App = AppMeta{
		Name:      app.Name,
		Guid:      app.Guid,
		SpaceName: app.SpaceData.Entity.Name,
		SpaceGuid: app.SpaceData.Meta.Guid,
		OrgName:   app.SpaceData.Entity.OrgData.Entity.Name,
		OrgGuid:   app.SpaceData.Entity.OrgData.Meta.Guid,
	}
}

func (r *appResponse) toStructs() (*AppMeta, error) {
	var empty cfclient.CloudFoundryError
	if r.Error != empty {
		// Wrapping the error so cfclient.IsAppNotFoundError can identify it
		return nil, errors.Wrap(r.Error, r.ErrorMessage)
	}
	if len(r.ErrorMessage) > 0 {
		return nil, errors.New(r.ErrorMessage)
	}
	return &r.App, nil
}

// fetchApp uses the cfClient to retrieve an App entity and
// stores it in the internal cache
func (c *clientCacheWrap) fetchAppByGuid(guid string) (*AppMeta, error) {
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
func (c *clientCacheWrap) GetAppByGuid(guid string) (*AppMeta, error) {
	var resp appResponse
	err := c.cache.Get(guid, &resp)
	if err != nil {
		return c.fetchAppByGuid(guid)
	}
	return resp.toStructs()
}

// Close release resources associated with this client
func (c *clientCacheWrap) Close() error {
	err := c.cache.Close()
	if err != nil {
		return fmt.Errorf("closing cache: %w", err)
	}
	return nil
}

// sanitizeCacheName returns a unique string that can be used safely as part of a file name
func sanitizeCacheName(name string) string {
	hash := sha1.Sum([]byte(name))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
