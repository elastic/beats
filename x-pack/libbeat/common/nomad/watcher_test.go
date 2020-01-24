// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

// "github.com/hashicorp/nomad/nomad/mock"
import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/elastic/beats/libbeat/tests/resources"
	api "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/stretchr/testify/assert"
)

const (
	NomadIndexHeader = "X-Nomad-Index"
	DefaultWaitIndex = 1
)

func TestWatcherAddAllocation(t *testing.T) {
	node := mock.Node()
	alloc := mock.Alloc()

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal([]interface{}{node})
		if err != nil {
			t.Error(err)
		}

		w.Header().Add(NomadIndexHeader, fmt.Sprint(time.Now().Unix()))
		w.Write(payload)
	})

	mux.HandleFunc("/v1/node/", func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal([]interface{}{alloc})
		if err != nil {
			t.Error(err)
		}

		w.Header().Add(NomadIndexHeader, fmt.Sprint(time.Now().Unix()))
		w.Write(payload)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Unexpected requested detected")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	config := &api.Config{
		Address:    server.URL,
		HttpClient: server.Client(),
	}

	options := WatchOptions{
		RefreshInterval: 1 * time.Second,
		Node:            node.Name,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	addedAllocs := []api.Allocation{}
	updatedAllocs := []api.Allocation{}
	deletedAllocs := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			addedAllocs = append(addedAllocs, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updatedAllocs = append(updatedAllocs, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deletedAllocs = append(deletedAllocs, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, addedAllocs, 1)
	assert.Len(t, updatedAllocs, 0)
	assert.Len(t, deletedAllocs, 0)
}

func TestWatcherUnchangedIndex(t *testing.T) {
	node := mock.Node()
	alloc := mock.Alloc()

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal([]interface{}{node})
		if err != nil {
			t.Error(err)
		}

		w.Header().Add(NomadIndexHeader, fmt.Sprint(time.Now().Unix()))
		w.Write(payload)
	})

	mux.HandleFunc("/v1/node/", func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal([]interface{}{alloc})
		if err != nil {
			t.Error(err)
		}

		w.Header().Add(NomadIndexHeader, fmt.Sprint(DefaultWaitIndex))
		w.Write(payload)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Unexpected requested detected")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	config := &api.Config{
		Address:    server.URL,
		HttpClient: server.Client(),
	}

	options := WatchOptions{
		RefreshInterval: 1 * time.Second,
		Node:            node.Name,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	addedAllocs := []api.Allocation{}
	updatedAllocs := []api.Allocation{}
	deletedAllocs := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			addedAllocs = append(addedAllocs, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updatedAllocs = append(updatedAllocs, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deletedAllocs = append(deletedAllocs, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, addedAllocs, 0)
	assert.Len(t, updatedAllocs, 0)
	assert.Len(t, deletedAllocs, 0)
}
