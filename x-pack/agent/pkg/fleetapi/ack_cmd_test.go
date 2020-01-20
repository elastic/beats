// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAck(t *testing.T) {
	const withAPIKey = "secret"
	agentInfo := &agentinfo{}

	t.Run("Test ack roundtrip", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
{
    "action": "ack",
    "success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/acks", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)

				responses := struct {
					ActionIDs []string `json:"action_ids"`
				}{}

				decoder := json.NewDecoder(r.Body)
				defer r.Body.Close()

				err := decoder.Decode(&responses)
				require.NoError(t, err)

				require.Equal(t, 1, len(responses.ActionIDs))

				id := responses.ActionIDs[0]
				require.Equal(t, "my-id", id)

				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			action := &ActionPolicyChange{
				ActionID:   "my-id",
				ActionType: "POLICY_CHANGE",
				Policy: map[string]interface{}{
					"id": "policy_id",
				},
			}

			cmd := NewAckCmd(&agentinfo{}, client)

			request := AckRequest{
				Actions: []string{
					action.ID(),
				},
			}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)
			require.Equal(t, "ack", r.Action)
		},
	))
}
