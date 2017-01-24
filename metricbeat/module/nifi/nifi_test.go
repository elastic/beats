package nifi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsCluster tests the IsCluster() function
func TestIsCluster(t *testing.T) {
	host := GetEnvHost()
	port := GetEnvPort()
	host = fmt.Sprintf("%s:%s", host, port)
	client := &http.Client{}

	result := IsCluster(host, client)

	assert.Equal(t, true, result)
}

func TestGetNodeMap(t *testing.T) {
	host := GetEnvHost()
	port := GetEnvPort()
	host = fmt.Sprintf("%s:%s", host, port)
	client := &http.Client{}

	result, err := GetNodeMap(host, client)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))

	// try to access the nodeID in the map using the host string as indexi w
	if val, ok := result[host]; ok {
		assert.True(t, len([]rune(val)) > 0)
	} else {
		assert.Fail(t, "Key with hostname does not exist")
	}
}
