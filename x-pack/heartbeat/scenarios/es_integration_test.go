package scenarios

import (
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStates(t *testing.T) {
	esc, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"127.0.0.1:9200"},
		Username:  "elastic",
		Password:  "changeme",
	})
	require.NoError(t, err)
	h, err := esc.Cluster.Health()
	require.NoError(t, err)
	fmt.Printf("BODY %v", h.Body)
}
