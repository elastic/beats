package fleetapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: We should keep track the time the action was created in our system, I have a feeling we might have
// some lag or thing to discover from that.
func skip(t *testing.T) {
	t.SkipNow()
}

func TestCheckin(t *testing.T) {
	const agentID = "bob"
	const withAccessToken = "secret"

	t.Run("Send back status of actions", skip)
	t.Run("Propagate any errors from the server", skip)
	t.Run("Checkin receives a PolicyChange", skip)
	t.Run("Checkin receives known and unknown action type", withServerWithAuthClient(
		func(t *testing.T) *http.ServeMux {
			raw := `
{
  "actions": [{
    "type": "POLICY_CHANGE",
    "id": "id1",
    "data": {
      "policy":  {
       "id": "policy-id",
        "outputs": {
          "default": {
            "hosts": "https://localhost:9200"
          }
        },
        "streams": [
            "id": "string",
            "type": "logs",
						"path": "/var/log/hello.log",
            "output": {
              "use_output": "default"
            }
          }
        ]
      }
    }
  },
	{
	"type": "WHAT_TO_DO_WITH_IT",
	"id": "id2"
	}],
	"success": true
}
`
			mux := http.NewServeMux()
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentID)
			mux.HandleFunc(path, accessTokenHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAccessToken))
			return mux
		}, withAccessToken,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentID, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

			require.Equal(t, 2, len(r.Actions))

			// PolicyChangeAction
			require.Equal(t, "id1", r.Actions[0].ID())
			require.Equal(t, "POLICY_CHANGE", r.Actions[0].Type())

			// UnknownAction
			require.Equal(t, "id2", r.Actions[1].ID())
			require.Equal(t, "UNKNOWN", r.Actions[1].Type())
			require.Equal(t, "WHAT_TO_DO_WITH_IT", r.Actions[1].(*UnknownAction).OriginalType())
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
			path := fmt.Sprintf("/api/fleet/agents/%s/checkin", agentID)
			mux.HandleFunc(path, accessTokenHandler(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, raw)
			}, withAccessToken))
			return mux
		}, withAccessToken,
		func(t *testing.T, client clienter) {
			cmd := NewCheckinCmd(agentID, client)

			request := CheckinRequest{}

			r, err := cmd.Execute(&request)
			require.NoError(t, err)
			require.True(t, r.Success)

			require.Equal(t, 0, len(r.Actions))
		},
	))
}
