package elasticsearch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClientConnect(t *testing.T) {
	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)

	assert.Nil(t, err)
	assert.True(t, client.IsConnected())
}
