package loggregator

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
)

// IngressOption is the type of a configurable client option.
type IngressOption func(*IngressClient)

func WithDialOptions(opts ...grpc.DialOption) IngressOption {
	return func(c *IngressClient) {
		c.dialOpts = append(c.dialOpts, opts...)
	}
}

// WithTag allows for the configuration of arbitrary string value
// metadata which will be included in all data sent to Loggregator
func WithTag(name, value string) IngressOption {
	return func(c *IngressClient) {
		c.tags[name] = value
	}
}

// WithBatchMaxSize allows for the configuration of the number of messages to
// collect before emitting them into loggregator. By default, its value is 100
// messages.
//
// Note that aside from batch size, messages will be flushed from
// the client into loggregator at a fixed interval to ensure messages are not
// held for an undue amount of time before being sent. In other words, even if
// the client has not yet achieved the maximum batch size, the batch interval
// may trigger the messages to be sent.
func WithBatchMaxSize(maxSize uint) IngressOption {
	return func(c *IngressClient) {
		c.batchMaxSize = maxSize
	}
}

// WithBatchFlushInterval allows for the configuration of the maximum time to
// wait before sending a batch of messages. Note that the batch interval
// may be triggered prior to the batch reaching the configured maximum size.
func WithBatchFlushInterval(d time.Duration) IngressOption {
	return func(c *IngressClient) {
		c.batchFlushInterval = d
	}
}

// WithAddr allows for the configuration of the loggregator v2 address.
// The value to defaults to localhost:3458, which happens to be the default
// address in the loggregator server.
func WithAddr(addr string) IngressOption {
	return func(c *IngressClient) {
		c.addr = addr
	}
}

// Logger declares the minimal logging interface used within the v2 client
type Logger interface {
	Printf(string, ...interface{})
	Panicf(string, ...interface{})
}

// WithLogger allows for the configuration of a logger.
// By default, the logger is disabled.
func WithLogger(l Logger) IngressOption {
	return func(c *IngressClient) {
		c.logger = l
	}
}

// WithContext configures the context that manages the lifecycle for the gRPC
// connection. It defaults to a context.Background().
func WithContext(ctx context.Context) IngressOption {
	return func(c *IngressClient) {
		c.ctx = ctx
	}
}

// IngressClient represents an emitter into loggregator. It should be created with the
// NewIngressClient constructor.
type IngressClient struct {
	client loggregator_v2.IngressClient
	sender loggregator_v2.Ingress_BatchSenderClient

	envelopes chan *loggregator_v2.Envelope
	tags      map[string]string

	batchMaxSize       uint
	batchFlushInterval time.Duration
	addr               string

	dialOpts []grpc.DialOption

	logger Logger

	closeErrors chan error

	ctx    context.Context
	cancel func()
}

// NewIngressClient creates a v2 loggregator client. Its TLS configuration
// must share a CA with the loggregator server.
func NewIngressClient(tlsConfig *tls.Config, opts ...IngressOption) (*IngressClient, error) {
	c := &IngressClient{
		envelopes:          make(chan *loggregator_v2.Envelope, 100),
		tags:               make(map[string]string),
		batchMaxSize:       100,
		batchFlushInterval: 100 * time.Millisecond,
		addr:               "localhost:3458",
		logger:             log.New(ioutil.Discard, "", 0),
		closeErrors:        make(chan error),
		ctx:                context.Background(),
	}

	for _, o := range opts {
		o(c)
	}

	c.ctx, c.cancel = context.WithCancel(c.ctx)

	c.dialOpts = append(c.dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	conn, err := grpc.Dial(
		c.addr,
		c.dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	c.client = loggregator_v2.NewIngressClient(conn)

	go c.startSender()

	return c, nil
}

// protoEditor is required for v1 envelopes. It should be removed once v1
// is removed. It is necessary to prevent any v1 dependency in the v2 path.
type protoEditor interface {
	SetLogAppInfo(appID, sourceType, sourceInstance string)
	SetGaugeAppInfo(appID string, index int)
	SetCounterAppInfo(appID string, index int)
	SetSourceInfo(sourceID, instanceID string)
	SetLogToStdout()
	SetGaugeValue(name string, value float64, unit string)
	SetDelta(d uint64)
	SetTotal(t uint64)
	SetTag(name, value string)
}

// EmitLogOption is the option type passed into EmitLog
type EmitLogOption func(proto.Message)

// WithAppInfo configures the meta data associated with emitted data. Exists
// for backward compatability. If possible, use WithSourceInfo instead.
func WithAppInfo(appID, sourceType, sourceInstance string) EmitLogOption {
	return WithSourceInfo(appID, sourceType, sourceInstance)
}

// WithSourceInfo configures the meta data associated with emitted data
func WithSourceInfo(sourceID, sourceType, sourceInstance string) EmitLogOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.SourceId = sourceID
			e.InstanceId = sourceInstance
			e.Tags["source_type"] = sourceType
		case protoEditor:
			e.SetLogAppInfo(sourceID, sourceType, sourceInstance)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// WithStdout sets the output type to stdout. Without using this option,
// all data is assumed to be stderr output.
func WithStdout() EmitLogOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.GetLog().Type = loggregator_v2.Log_OUT
		case protoEditor:
			e.SetLogToStdout()
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// EmitLog sends a message to loggregator.
func (c *IngressClient) EmitLog(message string, opts ...EmitLogOption) {
	e := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Log{
			Log: &loggregator_v2.Log{
				Payload: []byte(message),
				Type:    loggregator_v2.Log_ERR,
			},
		},
		Tags: make(map[string]string),
	}

	for k, v := range c.tags {
		e.Tags[k] = v
	}

	for _, o := range opts {
		o(e)
	}

	c.envelopes <- e
}

// EmitGaugeOption is the option type passed into EmitGauge.
type EmitGaugeOption func(proto.Message)

// WithGaugeAppInfo configures an envelope with both the app ID and index.
// Exists for backward compatability. If possible, use WithGaugeSourceInfo
// instead.
func WithGaugeAppInfo(appID string, index int) EmitGaugeOption {
	return WithGaugeSourceInfo(appID, strconv.Itoa(index))
}

// WithGaugeSourceInfo configures an envelope with both the source ID and
// instance ID.
func WithGaugeSourceInfo(sourceID, instanceID string) EmitGaugeOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.SourceId = sourceID
			e.InstanceId = instanceID
		case protoEditor:
			e.SetSourceInfo(sourceID, instanceID)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// WithGaugeValue adds a gauge information. For example,
// to send information about current CPU usage, one might use:
//
// WithGaugeValue("cpu", 3.0, "percent")
//
// An number of calls to WithGaugeValue may be passed into EmitGauge.
// If there are duplicate names in any of the options, i.e., "cpu" and "cpu",
// then the last EmitGaugeOption will take precedence.
func WithGaugeValue(name string, value float64, unit string) EmitGaugeOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.GetGauge().Metrics[name] = &loggregator_v2.GaugeValue{Value: value, Unit: unit}
		case protoEditor:
			e.SetGaugeValue(name, value, unit)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// EmitGauge sends the configured gauge values to loggregator.
// If no EmitGaugeOption values are present, the client will emit
// an empty gauge.
func (c *IngressClient) EmitGauge(opts ...EmitGaugeOption) {
	e := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: make(map[string]*loggregator_v2.GaugeValue),
			},
		},
		Tags: make(map[string]string),
	}

	for k, v := range c.tags {
		e.Tags[k] = v
	}

	for _, o := range opts {
		o(e)
	}

	c.envelopes <- e
}

// EmitCounterOption is the option type passed into EmitCounter.
type EmitCounterOption func(proto.Message)

// WithDelta is an option that sets the delta for a counter.
func WithDelta(d uint64) EmitCounterOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.GetCounter().Delta = d
		case protoEditor:
			e.SetDelta(d)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// WithTotal is an option that sets the total for a counter.
func WithTotal(t uint64) EmitCounterOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.GetCounter().Total = t
			e.GetCounter().Delta = 0
		case protoEditor:
			e.SetTotal(t)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// WithCounterAppInfo configures an envelope with both the app ID and index.
// Exists for backward compatability. If possible, use WithCounterSourceInfo
// instead.
func WithCounterAppInfo(appID string, index int) EmitCounterOption {
	return WithCounterSourceInfo(appID, strconv.Itoa(index))
}

// WithCounterSourceInfo configures an envelope with both the app ID and
// source ID.
func WithCounterSourceInfo(sourceID, instanceID string) EmitCounterOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.SourceId = sourceID
			e.InstanceId = instanceID
		case protoEditor:
			e.SetSourceInfo(sourceID, instanceID)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// EmitCounter sends a counter envelope with a delta of 1.
func (c *IngressClient) EmitCounter(name string, opts ...EmitCounterOption) {
	e := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Counter{
			Counter: &loggregator_v2.Counter{
				Name:  name,
				Delta: uint64(1),
			},
		},
		Tags: make(map[string]string),
	}

	for k, v := range c.tags {
		e.Tags[k] = v
	}

	for _, o := range opts {
		o(e)
	}

	c.envelopes <- e
}

// EmitTimerOption is the option type passed into EmitTimer.
type EmitTimerOption func(proto.Message)

// WithTimerSourceInfo configures an envelope with both the source and instance
// IDs.
func WithTimerSourceInfo(sourceID, instanceID string) EmitTimerOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.SourceId = sourceID
			e.InstanceId = instanceID
		case protoEditor:
			e.SetSourceInfo(sourceID, instanceID)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// EmitTimer sends a timer envelope with the given name, start time and stop time.
func (c *IngressClient) EmitTimer(name string, start, stop time.Time, opts ...EmitTimerOption) {
	e := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Timer{
			Timer: &loggregator_v2.Timer{
				Name:  name,
				Start: start.UnixNano(),
				Stop:  stop.UnixNano(),
			},
		},
		Tags: make(map[string]string),
	}

	for k, v := range c.tags {
		e.Tags[k] = v
	}

	for _, o := range opts {
		o(e)
	}

	c.envelopes <- e
}

// EmitEventOption is the option type passed into EmitEvent.
type EmitEventOption func(proto.Message)

// WithEventSourceInfo configures an envelope with both the source and instance
// IDs.
func WithEventSourceInfo(sourceID, instanceID string) EmitEventOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.SourceId = sourceID
			e.InstanceId = instanceID
		case protoEditor:
			e.SetSourceInfo(sourceID, instanceID)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// EmitEvent sends an Event envelope.
func (c *IngressClient) EmitEvent(ctx context.Context, title, body string, opts ...EmitEventOption) error {
	e := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		Message: &loggregator_v2.Envelope_Event{
			Event: &loggregator_v2.Event{
				Title: title,
				Body:  body,
			},
		},
		Tags: make(map[string]string),
	}

	for k, v := range c.tags {
		e.Tags[k] = v
	}

	for _, o := range opts {
		o(e)
	}

	_, err := c.client.Send(ctx, &loggregator_v2.EnvelopeBatch{
		Batch: []*loggregator_v2.Envelope{e},
	})

	return err
}

// Emit sends an envelope. It will sent within a batch.
func (c *IngressClient) Emit(e *loggregator_v2.Envelope) {
	c.envelopes <- e
}

// CloseSend will flush the envelope buffers and close the stream to the
// ingress server. This method will block until the buffers are flushed.
func (c *IngressClient) CloseSend() error {
	close(c.envelopes)

	return <-c.closeErrors
}

func (c *IngressClient) startSender() {
	defer c.cancel()

	t := time.NewTimer(c.batchFlushInterval)

	var batch []*loggregator_v2.Envelope
	for {
		select {
		case env, ok := <-c.envelopes:
			if !ok {
				if len(batch) > 0 {
					err := c.flush(batch)
					c.closeAndRecv()
					c.closeErrors <- err
					return
				}

				c.closeAndRecv()
				c.closeErrors <- nil

				return
			}

			batch = append(batch, env)

			if len(batch) >= int(c.batchMaxSize) {
				c.flush(batch)
				batch = nil
				if !t.Stop() {
					<-t.C
				}
				t.Reset(c.batchFlushInterval)
			}
		case <-t.C:
			if len(batch) > 0 {
				c.flush(batch)
				batch = nil
			}
			t.Reset(c.batchFlushInterval)
		}
	}
}

func (c *IngressClient) closeAndRecv() {
	if c.sender == nil {
		return
	}
	c.sender.CloseAndRecv()
}

func (c *IngressClient) flush(batch []*loggregator_v2.Envelope) error {
	err := c.emit(batch)
	if err != nil {
		c.logger.Printf("Error while flushing: %s", err)
	}

	return err
}

func (c *IngressClient) emit(batch []*loggregator_v2.Envelope) error {
	if c.sender == nil {
		var err error
		c.sender, err = c.client.BatchSender(c.ctx)
		if err != nil {
			return err
		}
	}

	err := c.sender.Send(&loggregator_v2.EnvelopeBatch{Batch: batch})
	if err != nil {
		c.sender = nil
		return err
	}

	return nil
}

// WithEnvelopeTag adds a tag to the envelope.
func WithEnvelopeTag(name, value string) func(proto.Message) {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.Tags[name] = value
		case protoEditor:
			e.SetTag(name, value)
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

// WithEnvelopeTags adds tag information that can be text, integer, or decimal to
// the envelope.  WithEnvelopeTags expects a single call with a complete map
// and will overwrite if called a second time.
func WithEnvelopeTags(tags map[string]string) func(proto.Message) {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			for name, value := range tags {
				e.Tags[name] = value
			}
		case protoEditor:
			for name, value := range tags {
				e.SetTag(name, value)
			}
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}
