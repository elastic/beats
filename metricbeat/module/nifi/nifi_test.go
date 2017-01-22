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
	fmt.Printf("%v", result)
}
