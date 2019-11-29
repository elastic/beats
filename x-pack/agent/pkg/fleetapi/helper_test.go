package fleetapi

import (
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/config"
)

func accessTokenHandler(handler http.HandlerFunc, accessToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("kbn-fleet-access-token") != accessToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

func withServer(m func(t *testing.T) *http.ServeMux, test func(t *testing.T, host string)) func(t *testing.T) {
	return func(t *testing.T) {
		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		go http.Serve(listener, m(t))

		test(t, "localhost:"+strconv.Itoa(port))
	}
}

func withServerWithAuthClient(
	m func(t *testing.T) *http.ServeMux,
	accessToken string,
	test func(t *testing.T, client clienter),
) func(t *testing.T) {

	return withServer(m, func(t *testing.T, host string) {
		cfg := config.MustNewConfigFrom(map[string]interface{}{
			"host": host,
		})
		client, err := NewAuthWithConfig(nil, cfg, accessToken)
		require.NoError(t, err)
		test(t, client)
	})
}
