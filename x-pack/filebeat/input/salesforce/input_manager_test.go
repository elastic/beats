// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func makeTestStore(data map[string]any) *statestore.Store {
	memstore := &storetest.MapStore{Table: data}
	reg := statestore.NewRegistry(&storetest.MemoryStore{
		Stores: map[string]*storetest.MapStore{
			"test": memstore,
		},
	})
	store, err := reg.Get("test")
	if err != nil {
		panic("failed to create test store")
	}
	return store
}

// compile-time check if stateStore implements statestore.States
var _ statestore.States = stateStore{}

type stateStore struct{}

func (stateStore) StoreFor(string) (*statestore.Store, error) {
	return makeTestStore(map[string]any{"hello": "world"}), nil
}
func (stateStore) StoreKey() string               { return "salesforce-test-store" }
func (stateStore) CleanupInterval() time.Duration { return time.Duration(0) }

func TestInputManager(t *testing.T) {
	inputManager := NewInputManager(logptest.NewTestingLogger(t, "salesforce_test"), stateStore{})
	t.Cleanup(inputManager.Close)

	config, err := conf.NewConfigFrom(map[string]any{
		"url":     "https://salesforce.com",
		"version": 46,
		"auth": &authConfig{
			OAuth2: &OAuth2{JWTBearerFlow: &JWTBearerFlow{
				Enabled:        new(true),
				URL:            "https://salesforce.com",
				ClientID:       "xyz",
				ClientUsername: "xyz",
				ClientKeyPath:  "xyz",
			}},
		},
		"event_monitoring_method": map[string]any{
			"object": map[string]any{
				"enabled":  true,
				"interval": "4ns",
				"query": map[string]any{
					"default": defaultLoginObjectQuery,
					"value":   valueLoginObjectQuery,
				},
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	})
	assert.NoError(t, err)

	_, err = inputManager.Create(config)
	assert.NoError(t, err)
}

func TestInputManagerRejectsInvalidConfigOnCreate(t *testing.T) {
	inputManager := NewInputManager(logptest.NewTestingLogger(t, "salesforce_test"), stateStore{})
	t.Cleanup(inputManager.Close)

	config, err := conf.NewConfigFrom(map[string]any{
		"url":     "https://salesforce.com",
		"version": 46,
		"auth": &authConfig{
			OAuth2: &OAuth2{JWTBearerFlow: &JWTBearerFlow{
				Enabled:        new(true),
				URL:            "https://salesforce.com",
				ClientID:       "xyz",
				ClientUsername: "xyz",
				ClientKeyPath:  "xyz",
			}},
		},
		"event_monitoring_method": map[string]any{
			"object": map[string]any{
				"enabled":  true,
				"interval": "4ns",
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = inputManager.Create(config)
	require.Error(t, err)
	assert.ErrorContains(t, err, `"event_monitoring_method.object.query" must be configured`)
}

func TestSource(t *testing.T) {
	want := "https://salesforce.com"
	src := source{cfg: config{URL: want}}
	got := src.Name()
	assert.Equal(t, want, got)
}
