package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/sarama"
	"github.com/elastic/sarama/mocks"
)

func TestClientOutputListener_customMock(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"hosts":   []string{"localhost:9094"},
		"topic":   "testTopic",
		"timeout": "1s",
	})
	require.NoError(t, err, "could not create config from map")
	reg := monitoring.NewRegistry()
	outGrup, err := makeKafka(
		nil,
		beat.Info{
			Beat:        "libbeat",
			IndexPrefix: "testbeat",
			Logger:      logger},
		outputs.NewStats(reg), cfg)
	if err != nil {
		t.Fatal(err)
	}

	c, ok := outGrup.Clients[0].(*client)
	require.Truef(t, ok, "Expected output to be of type %T", &client{})

	producer := &mockProducer{
		inCh:      make(chan *sarama.ProducerMessage, 2),
		successCh: make(chan *sarama.ProducerMessage, 1),
		errCh:     make(chan *sarama.ProducerError, 1),
		processInput: func(m *sarama.ProducerMessage) error {
			bs, err := m.Value.Encode()
			assert.NoError(t, err, "could not encode message")

			dic := map[string]any{}
			err = json.Unmarshal(bs, &dic)
			if err != nil {
				return fmt.Errorf("could not decode message: %w", err)
			}

			fmt.Println("data received: ", dic)

			if dic["to_drop"].(string) == "true" {
				return fmt.Errorf("to_drop == true, returning an error: %w",
					sarama.ErrInvalidMessage) // required for permanent error
			}
			return nil
		},
	}

	c.producer = producer
	c.wg.Add(2)
	go c.successWorker(c.producer.Successes())
	go c.errorWorker(c.producer.Errors())

	counter := &countListener{}
	observer := publisher.OutputListener{Listener: counter}
	b := pipeline.MockBatch{
		Mu: sync.Mutex{},
		EventList: []publisher.Event{
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":     "a message 1",
						"to_drop": "false"},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":     "a message 2",
						"to_drop": "true"},
					Private:    nil,
					TimeSeries: false,
				},
			},
		},
	}

	err = c.Publish(context.Background(), &b)
	require.NoError(t, err)

	// make the producer read the messages
	require.NoError(t, producer.run())
	require.NoError(t, producer.run())

	err = c.Close()
	require.NoError(t, err, "could not kafka client")

	assertOutputMetrics(t, counter, reg)
}

func assertOutputMetrics(t *testing.T, counter *countListener, reg *monitoring.Registry) {
	assert.Equal(t, int64(2), counter.new.Load())
	assert.Equal(t, int64(2), counter.acked.Load())
	assert.Equal(t, int64(1), counter.dropped.Load())

	metrics := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)
	evs, ok := metrics["events"]
	require.True(t, ok, "could not find 'events' in metrics")
	parsedEvs, ok := evs.(map[string]any)
	require.True(t, ok, "could not parse 'events' isn't map[string]int64, it's %T", evs)

	assert.Equal(t, int64(2), parsedEvs["total"].(int64))
	assert.Equal(t, int64(2), parsedEvs["acked"].(int64))
	assert.Equal(t, int64(1), parsedEvs["dropped"].(int64))
	assert.Equal(t, int64(1), parsedEvs["batches"].(int64))
}

func TestClientOutputListener_saramaMock(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "",
		// only print stacktrace for errors above ErrorLevel.
		zap.AddStacktrace(zapcore.ErrorLevel+1))

	cfgSarama := sarama.NewConfig()
	cfgSarama.Producer.Return.Successes = true
	cfgSarama.Producer.Return.Errors = true

	producer := mocks.NewAsyncProducer(t, cfgSarama)
	producer.ExpectInputAndSucceed()
	producer.ExpectInputAndFail(
		fmt.Errorf("test permanent error: %w", sarama.ErrInvalidMessage))

	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"hosts":   []string{"localhost:9094"},
		"topic":   "testTopic",
		"timeout": "1s",
	})
	require.NoError(t, err, "could not create config from map")

	reg := monitoring.NewRegistry()
	outGrup, err := makeKafka(
		nil,
		beat.Info{
			Beat:        "libbeat",
			IndexPrefix: "testbeat",
			Logger:      logger},
		outputs.NewStats(reg), cfg)
	require.NoError(t, err, "could not create kafka output")

	c, ok := outGrup.Clients[0].(*client)
	require.Truef(t, ok, "Expected output to be of type %T", &client{})

	c.producer = producer
	c.wg.Add(2)
	go c.successWorker(c.producer.Successes())
	go c.errorWorker(c.producer.Errors())

	counter := &countListener{}
	observer := publisher.OutputListener{Listener: counter}
	b := pipeline.MockBatch{
		Mu: sync.Mutex{},
		EventList: []publisher.Event{
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":     "message 1",
						"to_drop": "false"},
					Private:    nil,
					TimeSeries: false,
				},
			},
			{
				OutputListener: observer,
				Content: beat.Event{
					Timestamp: time.Time{},
					Meta:      nil,
					Fields: map[string]interface{}{
						"msg":     "message 2",
						"to_drop": "true"},
					Private:    nil,
					TimeSeries: false,
				},
			},
		},
	}

	err = c.Publish(context.Background(), &b)
	require.NoError(t, err, "could not publish batch")

	require.NoError(t, c.Close(), "failed closing kafka client")

	assertOutputMetrics(t, counter, reg)
}

var _ beat.OutputListener = (*countListener)(nil)

type countListener struct {
	acked,
	deadLetter,
	dropped,
	duplicateEvents,
	errTooMany,
	new,
	retryableErrors atomic.Int64
}

func (c *countListener) Acked() {
	c.acked.Add(1)
}

func (c *countListener) DeadLetter() {
	c.deadLetter.Add(1)
}

func (c *countListener) Dropped() {
	c.dropped.Add(1)
}

func (c *countListener) DuplicateEvents() {
	c.duplicateEvents.Add(1)
}

func (c *countListener) ErrTooMany() {
	c.errTooMany.Add(1)
}

func (c *countListener) NewEvent() {
	c.new.Add(1)
}

func (c *countListener) RetryableError() {
	c.retryableErrors.Add(1)
}

func (c *countListener) String() string {
	return fmt.Sprintf("New: %d, Acked: %d, Dropped: %d, DeadLetter: %d",
		c.new.Load(), c.acked.Load(), c.dropped.Load(), c.deadLetter.Load())
}

type mockProducer struct {
	wg   sync.WaitGroup
	once sync.Once

	inCh         chan *sarama.ProducerMessage
	successCh    chan *sarama.ProducerMessage
	errCh        chan *sarama.ProducerError
	processInput func(*sarama.ProducerMessage) error
}

func (m *mockProducer) run() error {
	m.wg.Add(1)
	defer m.wg.Done()

	msg, ok := <-m.inCh
	if !ok {
		return errors.New("producer already closed")
	}
	err := m.processInput(msg)
	if err != nil {
		m.errCh <- &sarama.ProducerError{
			Err: err,
			Msg: msg,
		}
	} else {
		m.successCh <- msg
	}

	return nil
}

func (m *mockProducer) AsyncClose() {
	_ = m.Close()
}

func (m *mockProducer) Close() error {
	m.once.Do(func() {
		close(m.inCh)
		close(m.successCh)
		close(m.errCh)
	})

	m.wg.Wait()

	return nil
}

func (m *mockProducer) Input() chan<- *sarama.ProducerMessage {
	return m.inCh
}

func (m *mockProducer) Successes() <-chan *sarama.ProducerMessage {
	return m.successCh
}

func (m *mockProducer) Errors() <-chan *sarama.ProducerError {
	return m.errCh
}

func (m *mockProducer) IsTransactional() bool {
	// TODO implement me
	panic("implement me")
}

func (m *mockProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	// TODO implement me
	panic("implement me")
}

func (m *mockProducer) BeginTxn() error {
	// TODO implement me
	panic("implement me")
}

func (m *mockProducer) CommitTxn() error {
	// TODO implement me
	panic("implement me")
}

func (m *mockProducer) AbortTxn() error {
	// TODO implement me
	panic("implement me")
}

func (m *mockProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	// TODO implement me
	panic("implement me")
}
