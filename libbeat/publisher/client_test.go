// +build !integration

package publisher

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that the correct client type is returned based on the given
// ClientOptions.
func TestGetClient(t *testing.T) {
	c := &client{
		publisher: &BeatPublisher{},
	}
	c.publisher.pipelines.async = &asyncPipeline{}
	c.publisher.pipelines.sync = &syncPipeline{}

	asyncClient := c.publisher.pipelines.async
	syncClient := c.publisher.pipelines.sync
	guaranteedClient := asyncClient
	guaranteedSyncClient := syncClient

	var testCases = []struct {
		in  []ClientOption
		out pipeline
	}{
		// Add new client options here:
		{[]ClientOption{}, asyncClient},
		{[]ClientOption{Sync}, syncClient},
		{[]ClientOption{Guaranteed}, guaranteedClient},
		{[]ClientOption{Guaranteed, Sync}, guaranteedSyncClient},
	}

	for _, test := range testCases {
		expected := reflect.ValueOf(test.out)
		_, client := c.getPipeline(test.in)
		actual := reflect.ValueOf(client)
		assert.Equal(t, expected.Pointer(), actual.Pointer())
	}
}
