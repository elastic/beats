// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	workerTestDefaultTimeout = 10 * time.Second
)

func TestMakeClientWorker(t *testing.T) {
	workQueue := make(chan *Work, 1)
	t.Cleanup(func() { close(workQueue) })
	client := newMockClient(func(batch publisher.Batch) error { return nil })

	w := MakeClientWorker(workQueue, client, *logp.NewNopLogger())
	t.Cleanup(func() { _ = w.Close() })
	work := NewWork(&mockBatch{})
	workQueue <- work

	select {
	case err := <-work.Result():
		assert.NoError(t, err, "expected no error, got %v", err)
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	assert.Equal(t, 1, client.publishCalls())
	assert.NoError(t, w.Close())
	assert.Equal(t, 1, client.closeCalls())
}

func TestClientWorkerPublishError(t *testing.T) {
	expectedError := errors.New("publish error")
	workQueue := make(chan *Work, 1)
	t.Cleanup(func() { close(workQueue) })
	client := newMockClient(func(batch publisher.Batch) error { return expectedError })

	w := MakeClientWorker(workQueue, client, *logp.NewNopLogger())
	t.Cleanup(func() { _ = w.Close() })
	work := NewWork(&mockBatch{})
	workQueue <- work

	select {
	case err := <-work.Result():
		assert.Equal(t, expectedError, err)
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	assert.Equal(t, 1, client.publishCalls())
}

func TestMakeClientWorkerNetworkClient(t *testing.T) {
	workQueue := make(chan *Work, 1)
	t.Cleanup(func() { close(workQueue) })
	client := newMockNetworkClient(func(batch publisher.Batch) error { return nil })

	w := MakeClientWorker(workQueue, client, *logp.NewNopLogger())
	t.Cleanup(func() { _ = w.Close() })
	batch := &mockBatch{}
	work := NewWork(batch)

	// trigger the client connection
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.NoError(t, err)
		assert.Equal(t, 1, batch.cancelledCalls())
		assert.Equal(t, 0, client.publishCalls())
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	// Wait until the connect call has been made
	waitUntilTrue(workerTestDefaultTimeout, func() bool {
		return client.connectCalls() == 1
	})

	// 2nd it should publish the batch
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.NoError(t, err)
		assert.Equal(t, 1, client.publishCalls())
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	assert.NoError(t, w.Close())
	assert.Equal(t, 1, client.closeCalls())
}

func TestNetworkClientPublishError(t *testing.T) {
	expectedError := errors.New("publish error")
	workQueue := make(chan *Work, 1)
	t.Cleanup(func() { close(workQueue) })
	client := newMockNetworkClient(func(batch publisher.Batch) error { return expectedError })

	w := MakeClientWorker(workQueue, client, *logp.NewNopLogger())
	t.Cleanup(func() { _ = w.Close() })
	batch := &mockBatch{}
	work := NewWork(batch)

	// trigger the client connection
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.NoError(t, err)
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	// Wait until the connect call has been made
	require.True(t, waitUntilTrue(workerTestDefaultTimeout, func() bool {
		return client.connectCalls() == 1
	}))

	// Check if the publishBatch error is returned
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.Equal(t, 1, client.publishCalls())
		assert.Equal(t, expectedError, errors.Unwrap(err))
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	// trigger the client connection again (due to publish error)
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.NoError(t, err)
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	// Wait until the re-connect call has been made
	require.True(t, waitUntilTrue(workerTestDefaultTimeout, func() bool {
		return client.connectCalls() == 2
	}))

	assert.Equal(t, 2, batch.cancelledCalls())
	assert.Equal(t, 1, client.publishCalls())
}

func TestNetworkClientConnectError(t *testing.T) {
	workQueue := make(chan *Work, 1)
	t.Cleanup(func() { close(workQueue) })
	client := &mockNetworkClient{
		mockClient: &mockClient{},
		connectFn: func() error {
			return errors.New("connect error")
		},
	}

	w := MakeClientWorker(workQueue, client, *logp.NewNopLogger())
	t.Cleanup(func() { _ = w.Close() })
	batch := &mockBatch{}
	work := NewWork(batch)

	// trigger the client connection
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.NoError(t, err)
		assert.Equal(t, 1, batch.cancelledCalls())
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	// Wait until the connect call has been made
	require.True(t, waitUntilTrue(workerTestDefaultTimeout, func() bool {
		return client.connectCalls() == 1
	}))

	// should try to re-connect
	workQueue <- work
	select {
	case err := <-work.Result():
		assert.NoError(t, err)
		assert.Equal(t, 2, batch.cancelledCalls())
	case <-time.After(workerTestDefaultTimeout):
		t.Fatal("Test timed out")
	}

	// Wait until the connect call has been made
	require.True(t, waitUntilTrue(workerTestDefaultTimeout, func() bool {
		return client.connectCalls() == 2
	}))

	assert.Equal(t, 2, client.connectCalls())
	assert.Equal(t, 0, client.publishCalls())
}

type mockBatch struct {
	cancelledCount atomic.Int32
	events         []publisher.Event
}

func (m *mockBatch) ACK() {}

func (m *mockBatch) Drop() {}

func (m *mockBatch) Retry() {}

func (m *mockBatch) RetryEvents([]publisher.Event) {}

func (m *mockBatch) Events() []publisher.Event {
	return m.events
}

func (m *mockBatch) SplitRetry() bool {
	return false
}

func (m *mockBatch) Cancelled() {
	m.cancelledCount.Add(1)
}

func (m *mockBatch) cancelledCalls() int {
	return int(m.cancelledCount.Load())
}

type mockPublishFn func(publisher.Batch) error

func newMockClient(publishFn mockPublishFn) *mockClient {
	return &mockClient{publishFn: publishFn}
}

type mockClient struct {
	publishFn    mockPublishFn
	closeCount   atomic.Int32
	publishCount atomic.Int32
}

func (c *mockClient) closeCalls() int {
	return int(c.closeCount.Load())
}

func (c *mockClient) publishCalls() int {
	return int(c.publishCount.Load())
}

func (c *mockClient) String() string {
	return "mock_client"
}

func (c *mockClient) Close() error {
	c.closeCount.Add(1)
	return nil
}

func (c *mockClient) Publish(_ context.Context, batch publisher.Batch) error {
	c.publishCount.Add(1)
	if c.publishFn != nil {
		return c.publishFn(batch)
	}
	return nil
}

type mockNetworkClient struct {
	*mockClient
	connectCount atomic.Int32
	connectFn    func() error
}

func newMockNetworkClient(publishFn mockPublishFn) *mockNetworkClient {
	return &mockNetworkClient{mockClient: newMockClient(publishFn)}
}

func (c *mockNetworkClient) connectCalls() int {
	return int(c.connectCount.Load())
}

func (c *mockNetworkClient) Connect(_ context.Context) error {
	c.connectCount.Add(1)
	if c.connectFn != nil {
		return c.connectFn()
	}
	return nil
}

func waitUntilTrue(duration time.Duration, fn func() bool) bool {
	end := time.Now().Add(duration)
	for time.Now().Before(end) {
		if fn() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
