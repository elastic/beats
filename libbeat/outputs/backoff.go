package outputs

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/testing"
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

func (b *backoffClient) Test(d testing.Driver) {
	c, ok := b.client.(testing.Testable)
	if !ok {
		d.Fatal("output", errors.New("client doesn't support testing"))
	}

	c.Test(d)
}
