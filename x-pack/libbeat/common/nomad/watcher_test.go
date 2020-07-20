// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	api "github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/resources"
)

const (
	NomadIndexHeader = "X-Nomad-Index"
	DefaultWaitIndex = 1
)

func TestWatcherAddAllocation(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	alloc := api.Allocation{}
	alloc.ModifyIndex = 20
	alloc.CreateIndex = 20
	alloc.AllocModifyIndex = 20
	alloc.TaskGroup = "group1"
	alloc.ClientStatus = AllocClientStatusRunning

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
		RefreshInterval:  1 * time.Second,
		Node:             node.Name,
		InitialWaitIndex: 20,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 1)
	assert.Len(t, updated, 0)
	assert.Len(t, deleted, 0)
}

func TestWatcherUnchangedIndex(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	alloc := api.Allocation{}

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

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 0)
	assert.Len(t, updated, 0)
	assert.Len(t, deleted, 0)
}

func TestWatcherIgnoreOldAllocations(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	// The Watcher is initialized with an initial WaitIndex of 1
	// this allocation should be ignored
	alloc := api.Allocation{}
	alloc.ID = "9820bd24-6c67-013a-e0c3-6ce1129dc0d2"
	alloc.ModifyIndex = 0
	alloc.AllocModifyIndex = 0
	alloc.ClientStatus = AllocClientStatusRunning

	alloc1 := api.Allocation{}
	alloc1.ID = "5678ad24-6c67-013a-e0c3-6ce1129dc0d2"
	alloc1.ModifyIndex = 1
	alloc1.AllocModifyIndex = 1
	alloc1.ClientStatus = AllocClientStatusRunning

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
		payload, err := json.Marshal([]interface{}{alloc, alloc1})
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
		RefreshInterval:  1 * time.Second,
		Node:             node.Name,
		InitialWaitIndex: 1,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 0)
	assert.Len(t, updated, 1)
	assert.Len(t, deleted, 0)

	assert.NotEqual(t, updated[0].ID, alloc.ID)
}

func TestWatcherAddAllocationOnFirstRun(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	alloc := api.Allocation{}
	alloc.ModifyIndex = 72975148
	alloc.CreateIndex = 72636274
	alloc.AllocModifyIndex = 72975148
	alloc.ClientStatus = AllocClientStatusRunning

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
		RefreshInterval:  1 * time.Second,
		Node:             node.Name,
		InitialWaitIndex: 0,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 1)
	assert.Len(t, updated, 0)
	assert.Len(t, deleted, 0)
}

func TestWatcherUpdateAllocation(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	alloc := api.Allocation{}
	alloc.ModifyIndex = 20
	alloc.CreateIndex = 10
	alloc.AllocModifyIndex = 20
	alloc.ClientStatus = AllocClientStatusRunning

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
		RefreshInterval:  1 * time.Second,
		Node:             node.Name,
		InitialWaitIndex: 18, // not the initial run
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 0)
	assert.Len(t, updated, 1)
	assert.Len(t, deleted, 0)
}

func TestWatcherAllocationCreatedWhenChecked(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	// The Watcher is initialized with an initial WaitIndex of 1
	// this allocation should be ignored
	alloc := api.Allocation{}
	alloc.ID = "9820bd24-6c67-013a-e0c3-6ce1129dc0d2"
	alloc.ModifyIndex = 97
	alloc.AllocModifyIndex = 97
	alloc.CreateIndex = 85
	alloc.ClientStatus = api.AllocClientStatusRunning

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
		RefreshInterval:  1 * time.Second,
		Node:             node.Name,
		InitialWaitIndex: 85,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 1)
	assert.Len(t, updated, 0)
	assert.Len(t, deleted, 0)
}

func TestWatcherAllocationUpdatedWhenChecked(t *testing.T) {
	node := api.Node{}
	node.Name = "nomad1"

	// The Watcher is initialized with an initial WaitIndex of 1
	// this allocation should be ignored
	alloc := api.Allocation{}
	alloc.ID = "9820bd24-6c67-013a-e0c3-6ce1129dc0d2"
	alloc.ModifyIndex = 22286509
	alloc.AllocModifyIndex = 22286509
	alloc.CreateIndex = 22286479
	alloc.ClientStatus = api.AllocClientStatusRunning

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
		RefreshInterval:  1 * time.Second,
		Node:             node.Name,
		InitialWaitIndex: 22286509,
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Error(err)
	}

	watcher, err := NewWatcher(client, options)
	if err != nil {
		t.Error(err)
	}

	added := []api.Allocation{}
	updated := []api.Allocation{}
	deleted := []api.Allocation{}

	watcher.AddEventHandler(ResourceEventHandlerFuncs{
		AddFunc: func(alloc api.Allocation) {
			added = append(added, alloc)
		},
		UpdateFunc: func(alloc api.Allocation) {
			updated = append(updated, alloc)
		},
		DeleteFunc: func(alloc api.Allocation) {
			deleted = append(deleted, alloc)
		},
	})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	watcher.Start()
	defer watcher.Stop()

	assert.Len(t, added, 0)
	assert.Len(t, updated, 1)
	assert.Len(t, deleted, 0)
}
