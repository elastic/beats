// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cloudfoundry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/gofrs/uuid"
)

func TestClientCacheWrap(t *testing.T) {
	ttl := 500 * time.Millisecond
	guid := mustCreateFakeGuid()
	app := cfclient.App{
		Guid:   guid,
		Memory: 1, // use this field to track if from cache or from client
	}
	fakeClient := &fakeCFClient{app, 0}
	cache := newClientCacheWrap(fakeClient, ttl, logp.NewLogger("cloudfoundry"))

	// should err; different app client doesn't have
	_, err := cache.GetAppByGuid(mustCreateFakeGuid())
	assert.Error(t, err)

	// fetched from client for the first time
	one, err := cache.GetAppByGuid(guid)
	assert.NoError(t, err)
	assert.Equal(t, app, *one)
	assert.Equal(t, 1, fakeClient.callCount)

	// updated app in fake client, new fetch should not have updated app
	updatedApp := cfclient.App{
		Guid:   guid,
		Memory: 2,
	}
	fakeClient.app = updatedApp
	two, err := cache.GetAppByGuid(guid)
	assert.NoError(t, err)
	assert.Equal(t, app, *two)
	assert.Equal(t, 1, fakeClient.callCount)

	// wait the ttl, then it should have updated app
	time.Sleep(ttl)
	three, err := cache.GetAppByGuid(guid)
	assert.NoError(t, err)
	assert.Equal(t, updatedApp, *three)
	assert.Equal(t, 2, fakeClient.callCount)
}

type fakeCFClient struct {
	app       cfclient.App
	callCount int
}

func (f *fakeCFClient) GetAppByGuid(guid string) (cfclient.App, error) {
	if f.app.Guid != guid {
		return f.app, fmt.Errorf("no app with guid")
	}
	f.callCount++
	return f.app, nil
}

func mustCreateFakeGuid() string {
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return uuid.String()
}
