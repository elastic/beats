// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
)

type agentinfo struct{}

func (*agentinfo) AgentID() string { return "id" }

func TestCheckin(t *testing.T) {
	const withAPIKey = "secret"
	ctx := context.Background()
	agentInfo := &agentinfo{}

	t.Run("Propagate any errors from the server", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
	Something went wrong
	}
	`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client client.Sender) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			_, err := cmd.Execute(ctx, &request)
			require.Error(t, err)
		},
	))

	t.Run("Checkin receives a PolicyChange", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
	{
		"actions": [{
			"type": "POLICY_CHANGE",
			"id": "id1",
			"data": {
				"policy": {
					"id": "policy-id",
					"outputs": {
						"default": {
							"hosts": "https://localhost:9200"
						}
					},
					"datasources": [{
						"id": "string",
						"enabled": true,
						"use_output": "default",
						"inputs": [{
							"type": "logs",
							"streams": [{
								"paths": ["/var/log/hello.log"]
							}]
						}]
					}]
				}
			}
		}]
	}
	`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client client.Sender) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(ctx, &request)
			require.NoError(t, err)

			require.Equal(t, 1, len(r.Actions))

			// ActionPolicyChange
			require.Equal(t, "id1", r.Actions[0].ID())
			require.Equal(t, "POLICY_CHANGE", r.Actions[0].Type())
		},
	))

	t.Run("Checkin receives known and unknown action type", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
	{
	    "actions": [
	        {
	            "type": "POLICY_CHANGE",
	            "id": "id1",
	            "data": {
	                "policy": {
	                    "id": "policy-id",
	                    "outputs": {
	                        "default": {
	                            "hosts": "https://localhost:9200"
	                        }
	                    },
						"datasources": [{
							"id": "string",
							"enabled": true,
							"use_output": "default",
							"inputs": [{
								"type": "logs",
								"streams": [{
									"paths": ["/var/log/hello.log"]
								}]
							}]
						}]
	                }
	            }
	        },
	        {
	            "type": "WHAT_TO_DO_WITH_IT",
	            "id": "id2"
	        }
	    ]
	}
	`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client client.Sender) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(ctx, &request)
			require.NoError(t, err)

			require.Equal(t, 2, len(r.Actions))

			// ActionPolicyChange
			require.Equal(t, "id1", r.Actions[0].ID())
			require.Equal(t, "POLICY_CHANGE", r.Actions[0].Type())

			// UnknownAction
			require.Equal(t, "id2", r.Actions[1].ID())
			require.Equal(t, "UNKNOWN", r.Actions[1].Type())
			require.Equal(t, "WHAT_TO_DO_WITH_IT", r.Actions[1].(*ActionUnknown).OriginalType())
		},
	))

	t.Run("When we receive no action", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `{ "actions": [] }`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client client.Sender) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(ctx, &request)
			require.NoError(t, err)

			require.Equal(t, 0, len(r.Actions))
		},
	))

	t.Run("Meta are sent", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `{"actions": []}`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				type Request struct {
					Metadata *info.ECSMeta `json:"local_metadata"`
				}

				var req *Request

				content, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(content, &req))
				assert.Equal(t, "linux", req.Metadata.OS.Name)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client client.Sender) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{Metadata: testMetadata()}

			r, err := cmd.Execute(ctx, &request)
			require.NoError(t, err)

			require.Equal(t, 0, len(r.Actions))
		},
	))

	t.Run("No meta are sent when not provided", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `{"actions": []}`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				type Request struct {
					Metadata *info.ECSMeta `json:"local_metadata"`
				}

				var req *Request

				content, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(content, &req))
				assert.Nil(t, req.Metadata)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client client.Sender) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(ctx, &request)
			require.NoError(t, err)

			require.Equal(t, 0, len(r.Actions))
		},
	))
}
