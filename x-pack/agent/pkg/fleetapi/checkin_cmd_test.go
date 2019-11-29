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
	t.Run("Checkin receives no action to execute", skip)
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
            "api_token": "another-token",
            "id": "default",
            "name": "Default",
            "type": "elasticsearch",
            "url": "https://localhost:9200"
          }
        },
        "streams": [
          {
            "metricsets": [
              "container",
              "cpu"
            ],
            "id": "string",
            "type": "etc",
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
			require.Equal(t, "id1", r.Actions[0].ID())
			require.Equal(t, "id2", r.Actions[1].ID())

			fmt.Println(r.Actions[0].(*PolicyChangeAction).Policy)
		},
	))
}
