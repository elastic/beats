// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"context"
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
			raw := `{"action": "ack"}`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/ingest_manager/fleet/agents/%s/acks", agentInfo.AgentID())
			mux.HandleFunc(path, authHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)

				responses := struct {
					Events []AckEvent `json:"events"`
				}{}

				decoder := json.NewDecoder(r.Body)
				defer r.Body.Close()

				err := decoder.Decode(&responses)
				require.NoError(t, err)

				require.Equal(t, 1, len(responses.Events))

				id := responses.Events[0].ActionID
				require.Equal(t, "my-id", id)

				fmt.Fprintf(w, raw)
			}, withAPIKey))
			return mux
		}, withAPIKey,
		func(t *testing.T, client clienter) {
			action := &ActionConfigChange{
				ActionID:   "my-id",
				ActionType: "CONFIG_CHANGE",
				Config: map[string]interface{}{
					"id": "config_id",
				},
			}

			cmd := NewAckCmd(&agentinfo{}, client)

			request := AckRequest{
				Events: []AckEvent{
					{
						EventType: "ACTION_RESULT",
						SubType:   "ACKNOWLEDGED",
						ActionID:  action.ID(),
					},
				},
			}

			r, err := cmd.Execute(context.Background(), &request)
			require.NoError(t, err)
			require.Equal(t, "ack", r.Action)
		},
	))
}
