// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type agentinfo struct{}

func (*agentinfo) AgentID() string { return "id" }

func TestCheckin(t *testing.T) {
	const withAPIKey = "secret"
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
				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			_, err := cmd.Execute(&request)
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
	}],
	"success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

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
    ],
    "success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

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
			raw := `
{
  "actions": [],
	"success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

			require.Equal(t, 0, len(r.Actions))
		},
	))

	t.Run("Meta are sent", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
{
  "actions": [],
	"success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				type Request struct {
					Metadata map[string]interface{} `json:"local_metadata"`
				}
				req := &Request{}

				content, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(content, &req))

				assert.Equal(t, 1, len(req.Metadata))
				v, found := req.Metadata["key"]
				assert.True(t, found)

				intV, ok := v.(string)
				assert.True(t, ok)
				assert.Equal(t, "value", intV)

				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			meta := map[string]interface{}{
				"key": "value",
			}

			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{Metadata: meta}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

			require.Equal(t, 0, len(r.Actions))
		},
	))

	t.Run("No meta are sent when not provided", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
{
  "actions": [],
	"success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				req := make(map[string]interface{})

				content, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(content, &req))

				_, found := req["key"]
				assert.False(t, found)

				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentInfo, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

			require.Equal(t, 0, len(r.Actions))
		},
	))
}
