// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	api "github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/resources"
)

const (
	NomadIndexHeader = "X-Nomad-Index"
	DefaultWaitIndex = 1
)

func nomadRoutes(node api.Node, allocs []api.Allocation, waitIndex uint64) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal([]interface{}{node})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		w.Header().Add(NomadIndexHeader, fmt.Sprint(time.Now().Unix()))
		w.Write(payload)
	})

	mux.HandleFunc("/v1/node/", func(w http.ResponseWriter, r *http.Request) {
		payload, err := json.Marshal(allocs)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		w.Header().Add(NomadIndexHeader, fmt.Sprint(waitIndex))
		w.Write(payload)
	})

	return mux
}

type watcherEvents struct {
	added   []api.Allocation
	updated []api.Allocation
	deleted []api.Allocation
}

func (w *watcherEvents) testResourceEventHandler() ResourceEventHandlerFuncs {
	return ResourceEventHandlerFuncs{
		AddFunc:    func(alloc api.Allocation) { w.added = append(w.added, alloc) },
		UpdateFunc: func(alloc api.Allocation) { w.updated = append(w.updated, alloc) },
		DeleteFunc: func(alloc api.Allocation) { w.deleted = append(w.deleted, alloc) },
	}
}

func TestAllocationWatcher(t *testing.T) {
	tests := []struct {
		name             string
		node             api.Node
		allocs           []api.Allocation
		waitIndex        uint64
		initialWaitIndex uint64
		expected         watcherEvents
	}{
		{
			name: "allocation added",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 20, CreateIndex: 20,
					AllocModifyIndex: 20, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        400,
			initialWaitIndex: 20,
			expected: watcherEvents{
				added: []api.Allocation{
					{
						ModifyIndex: 20, CreateIndex: 20,
						AllocModifyIndex: 20, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				updated: nil,
				deleted: nil,
			},
		},
		{
			name: "ignore events due to unchanged WaitIndex",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 20, CreateIndex: 20,
					AllocModifyIndex: 20, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        DefaultWaitIndex,
			initialWaitIndex: DefaultWaitIndex,
			expected: watcherEvents{
				added:   nil,
				updated: nil,
				deleted: nil,
			},
		},
		{
			name: "ignore old allocations",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ID: "9820bd24-6c67-013a-e0c3-6ce1129dc0d2", ModifyIndex: 0,
					CreateIndex: 0, AllocModifyIndex: 0, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
				{
					ID: "5678ad24-6c67-013a-e0c3-6ce1129dc0d2", ModifyIndex: 1,
					CreateIndex: 0, AllocModifyIndex: 1, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        400,
			initialWaitIndex: 1,
			expected: watcherEvents{
				added: nil,
				updated: []api.Allocation{
					{
						ID: "5678ad24-6c67-013a-e0c3-6ce1129dc0d2", ModifyIndex: 1,
						CreateIndex: 0, AllocModifyIndex: 1, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				deleted: nil,
			},
		},
		{
			name: "on initial run all allocations are added",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 200, CreateIndex: 100,
					AllocModifyIndex: 200, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        400,
			initialWaitIndex: 0,
			expected: watcherEvents{
				added: []api.Allocation{
					{
						ModifyIndex: 200, CreateIndex: 100,
						AllocModifyIndex: 200, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				updated: nil,
				deleted: nil,
			},
		},
		{
			name: "allocation updated",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 20, CreateIndex: 10,
					AllocModifyIndex: 20, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        25,
			initialWaitIndex: 18,
			expected: watcherEvents{
				added: nil,
				updated: []api.Allocation{
					{
						ModifyIndex: 20, CreateIndex: 10,
						AllocModifyIndex: 20, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				deleted: nil,
			},
		},
		{
			name: "allocation created in the same index as the watcher check",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 97, CreateIndex: 85,
					AllocModifyIndex: 97, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        100,
			initialWaitIndex: 85,
			expected: watcherEvents{
				added: []api.Allocation{
					{
						ModifyIndex: 97, CreateIndex: 85,
						AllocModifyIndex: 97, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				updated: nil,
				deleted: nil,
			},
		},
		{
			name: "allocation updated in the same index as the watcher check",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 509, CreateIndex: 479,
					AllocModifyIndex: 509, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        600,
			initialWaitIndex: 509,
			expected: watcherEvents{
				added: nil,
				updated: []api.Allocation{
					{
						ModifyIndex: 509, CreateIndex: 479,
						AllocModifyIndex: 509, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				deleted: nil,
			},
		},
		{
			name: "old allocation index new modify index should be detected",
			node: api.Node{ID: uuid.Must(uuid.NewV4()).String(), Name: "nomad1"},
			allocs: []api.Allocation{
				{
					ModifyIndex: 20, CreateIndex: 11,
					AllocModifyIndex: 11, TaskGroup: "group1",
					NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
				},
			},
			waitIndex:        24,
			initialWaitIndex: 17,
			expected: watcherEvents{
				added: nil,
				updated: []api.Allocation{
					{
						ModifyIndex: 20, CreateIndex: 11,
						AllocModifyIndex: 11, TaskGroup: "group1",
						NodeName: "nomad1", ClientStatus: AllocClientStatusRunning,
					},
				},
				deleted: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := nomadRoutes(tt.node, tt.allocs, tt.waitIndex)
			server := httptest.NewServer(mux)
			defer server.Close()

			config := &api.Config{
				Address:    server.URL,
				HttpClient: server.Client(),
			}

			options := WatchOptions{
				RefreshInterval:  1 * time.Second,
				Node:             tt.node.Name,
				InitialWaitIndex: tt.initialWaitIndex,
			}

			client, err := api.NewClient(config)
			if err != nil {
				t.Error(err)
			}

			watcher, err := NewWatcher(client, options)
			if err != nil {
				t.Error(err)
			}

			events := watcherEvents{}
			watcher.AddEventHandler(events.testResourceEventHandler())

			goroutines := resources.NewGoroutinesChecker()
			defer goroutines.Check(t)

			watcher.Start()
			defer watcher.Stop()

			assert.Equal(t, tt.expected, events)
		})
	}
}
