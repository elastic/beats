package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/require"
)

func TestGetLicense(t *testing.T) {
	tests := map[string]struct {
		version        *common.Version
		resp           string
		expectedType   string
		expectedStatus string
	}{
		"v6_basic_active": {
			common.MustNewVersion("6.8.4"),
			`{"license": {"type": "basic", "status": "active"}}`,
			"basic",
			"active",
		},
		"v7_trial_active": {
			common.MustNewVersion("7.6.0"),
			`{"license": {"type": "trial", "status": "active"}}`,
			"trial",
			"active",
		},
		"v8_gold_expired": {
			common.MustNewVersion("8.0.0"),
			`{"license": {"type": "gold", "status": "expired"}}`,
			"gold",
			"expired",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var callIndex int
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				switch callIndex {
				case 0: // initial connection from client
					io.WriteString(rw, `{"version": {"number": "`+test.version.String()+`"}}`)
				case 1: // get license
					io.WriteString(rw, test.resp)
				}
				callIndex++
			}))
			defer server.Close()

			c, err := New(WithAddresses(server.URL))
			require.NoError(t, err)

			l, err := c.GetLicense()
			require.NoError(t, err)
			require.EqualValues(t, test.expectedStatus, l.Status)
			require.EqualValues(t, test.expectedType, l.Type)
		})
	}
}
