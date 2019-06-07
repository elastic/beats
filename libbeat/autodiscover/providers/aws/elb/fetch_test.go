package elb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newAPIFetcher(t *testing.T) {
	client := mockELBClient{}
	fetcher := newAPIFetcher(client)
	require.NotNil(t, fetcher)
}
