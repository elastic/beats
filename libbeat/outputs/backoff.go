package outputs

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

type backoffClient struct {
	client NetworkClient

	done    chan struct{}
	backoff *common.Backoff
}

// WithBackoff wraps a NetworkClient, adding exponential backoff support to a network client if connection/publishing failed.
func WithBackoff(client NetworkClient, init, max time.Duration) NetworkClient {
	done := make(chan struct{})
	backoff := common.NewBackoff(done, init, max)
	return &backoffClient{
		client:  client,
		done:    done,
		backoff: backoff,
	}
}

func (b *backoffClient) Connect() error {
	err := b.client.Connect()
	b.backoff.WaitOnError(err)
	return err
}

func (b *backoffClient) Close() error {
	err := b.client.Close()
	close(b.done)
	return err
}

func (b *backoffClient) Publish(batch publisher.Batch) error {
	err := b.client.Publish(batch)
	if err != nil {
		b.client.Close()
	}
	b.backoff.WaitOnError(err)
	return err
}

func (b *backoffClient) Client() NetworkClient {
	return b.client
}
