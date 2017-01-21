package modeutil

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/stretchr/testify/assert"
)

type dummyClient struct{}

func (dummyClient) Connect(timeout time.Duration) error { return nil }
func (dummyClient) Close() error                        { return nil }
func (dummyClient) PublishEvents(data []outputs.Data) (next []outputs.Data, err error) {
	return nil, nil
}
func (dummyClient) PublishEvent(data outputs.Data) error { return nil }

func makeTestClients(c map[string]interface{},
	newClient func(string) (mode.ProtocolClient, error),
) ([]mode.ProtocolClient, error) {
	cfg, err := common.NewConfigFrom(c)
	if err != nil {
		return nil, err
	}

	return MakeClients(cfg, newClient)
}

func TestMakeEmptyClientFail(t *testing.T) {
	config := map[string]interface{}{}
	clients, err := makeTestClients(config, dummyMockClientFactory)
	assert.Error(t, err)
	assert.Equal(t, 0, len(clients))
}

func TestMakeSingleClient(t *testing.T) {
	config := map[string]interface{}{
		"hosts": []string{"single"},
	}

	clients, err := makeTestClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(clients))
}

func TestMakeSingleClientWorkers(t *testing.T) {
	config := map[string]interface{}{
		"hosts":  []string{"single"},
		"worker": 3,
	}

	clients, err := makeTestClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(clients))
}

func TestMakeTwoClient(t *testing.T) {
	config := map[string]interface{}{
		"hosts": []string{"client1", "client2"},
	}

	clients, err := makeTestClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(clients))
}

func TestMakeTwoClientWorkers(t *testing.T) {
	config := map[string]interface{}{
		"hosts":  []string{"client1", "client2"},
		"worker": 3,
	}

	clients, err := makeTestClients(config, dummyMockClientFactory)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(clients))
}

func TestMakeTwoClientFail(t *testing.T) {
	config := map[string]interface{}{
		"hosts":  []string{"client1", "client2"},
		"worker": 3,
	}

	testError := errors.New("test")

	i := 1
	_, err := makeTestClients(config, func(host string) (mode.ProtocolClient, error) {
		if i%3 == 0 {
			return nil, testError
		}
		i++
		return dummyMockClientFactory(host)
	})
	assert.Equal(t, testError, err)
}

func dummyMockClientFactory(host string) (mode.ProtocolClient, error) {
	return dummyClient{}, nil
}
