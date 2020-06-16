package loggregator

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"log"
	"time"

	gendiodes "code.cloudfoundry.org/go-diodes"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// EnvelopeStreamConnector provides a way to connect to loggregator and
// consume a stream of envelopes. It handles reconnecting and provides
// a stream for the lifecycle of the given context. It should be created with
// the NewEnvelopeStreamConnector constructor.
type EnvelopeStreamConnector struct {
	addr    string
	tlsConf *tls.Config

	// Buffering
	bufferSize int
	alerter    func(int)

	log         Logger
	dialOptions []grpc.DialOption
}

// NewEnvelopeStreamConnector creates a new EnvelopeStreamConnector. Its TLS
// configuration must share a CA with the loggregator server.
func NewEnvelopeStreamConnector(
	addr string,
	t *tls.Config,
	opts ...EnvelopeStreamOption,
) *EnvelopeStreamConnector {

	c := &EnvelopeStreamConnector{
		addr:    addr,
		tlsConf: t,

		log: log.New(ioutil.Discard, "", 0),
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

// EnvelopeStreamOption configures a EnvelopeStreamConnector.
type EnvelopeStreamOption func(*EnvelopeStreamConnector)

// WithEnvelopeStreamLogger allows for the configuration of a logger.
// By default, the logger is disabled.
func WithEnvelopeStreamLogger(l Logger) EnvelopeStreamOption {
	return func(c *EnvelopeStreamConnector) {
		c.log = l
	}
}

// WithEnvelopeStreamConnectorDialOptions allows for configuration of
// grpc dial options.
func WithEnvelopeStreamConnectorDialOptions(opts ...grpc.DialOption) EnvelopeStreamOption {
	return func(c *EnvelopeStreamConnector) {
		c.dialOptions = opts
	}
}

// WithEnvelopeStreamBuffer enables the EnvelopeStream to read more quickly
// from the stream. It puts each envelope in a buffer that overwrites data if
// it is not being drained quick enough. If the buffer drops data, the
// 'alerter' function will be invoked with the number of envelopes dropped.
func WithEnvelopeStreamBuffer(size int, alerter func(missed int)) EnvelopeStreamOption {
	return func(c *EnvelopeStreamConnector) {
		c.bufferSize = size
		c.alerter = alerter
	}
}

// EnvelopeStream returns batches of envelopes. It blocks until its context
// is done or a batch of envelopes is available.
type EnvelopeStream func() []*loggregator_v2.Envelope

// Stream returns a new EnvelopeStream for the given context and request. The
// lifecycle of the EnvelopeStream is managed by the given context. If the
// underlying gRPC stream dies, it attempts to reconnect until the context
// is done.
func (c *EnvelopeStreamConnector) Stream(ctx context.Context, req *loggregator_v2.EgressBatchRequest) EnvelopeStream {
	s := newStream(ctx, c.addr, req, c.tlsConf, c.dialOptions, c.log)
	if c.alerter != nil || c.bufferSize > 0 {
		d := NewOneToOneEnvelopeBatch(
			c.bufferSize,
			gendiodes.AlertFunc(c.alerter),
			gendiodes.WithPollingContext(ctx),
		)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				d.Set(s.recv())
			}
		}()
		return d.Next
	}

	return s.recv
}

type stream struct {
	log    Logger
	ctx    context.Context
	req    *loggregator_v2.EgressBatchRequest
	client loggregator_v2.EgressClient
	rx     loggregator_v2.Egress_BatchedReceiverClient
}

func newStream(
	ctx context.Context,
	addr string,
	req *loggregator_v2.EgressBatchRequest,
	c *tls.Config,
	opts []grpc.DialOption,
	log Logger,
) *stream {
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	conn, err := grpc.Dial(
		addr,
		opts...,
	)
	if err != nil {
		// This error occurs on invalid configuration. And more notably,
		// it does NOT occur if the server is not up.
		log.Panicf("invalid gRPC dial configuration: %s", err)
	}

	// Protect against a go-routine leak. gRPC will keep a go-routine active
	// within the connection to keep the connectin alive. We have to close
	// this or the go-routine leaks. This is untested. We had trouble exposing
	// the underlying connectin was still active.
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	client := loggregator_v2.NewEgressClient(conn)

	return &stream{
		ctx:    ctx,
		req:    req,
		client: client,
		log:    log,
	}
}

func (s *stream) recv() []*loggregator_v2.Envelope {
	for {
		ok := s.connect(s.ctx)
		if !ok {
			return nil
		}
		batch, err := s.rx.Recv()
		if err != nil {
			s.rx = nil
			continue
		}

		return batch.Batch
	}
}

func (s *stream) connect(ctx context.Context) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		default:
			if s.rx != nil {
				return true
			}

			var err error
			s.rx, err = s.client.BatchedReceiver(
				ctx,
				s.req,
			)

			if err != nil {
				s.log.Printf("Error connecting to Logs Provider: %s", err)
				time.Sleep(50 * time.Millisecond)
				continue
			}

			return true
		}
	}
}
