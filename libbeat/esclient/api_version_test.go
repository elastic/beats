package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	v760 := common.MustNewVersion("7.6.0")

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		io.WriteString(rw, `{"version": {"number": "`+v760.String()+`"}}`)
	}))
	defer server.Close()

	c, err := New(WithAddresses(server.URL))
	require.NoError(t, err)
	require.EqualValues(t, v760, c.GetVersion())
}
