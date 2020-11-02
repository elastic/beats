package nsq

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/nsqio/go-nsq"
)

type client struct {
	log      *logp.Logger
	observer outputs.Observer
	outputs.NetworkClient
	codec codec.Codec
	// key   *fmtstr.EventFormatString
	index string

	// for nsq
	nsqd  string
	topic string
	// topic    outil.Selector
	producer *nsq.Producer
	// config   *nsq.Config
	config *nsq.Config

	// producer sarama.AsyncProducer
	mux sync.Mutex
	wg  sync.WaitGroup
}

type msgRef struct {
	client *client
	count  int32
	total  int
	failed []publisher.Event
	batch  publisher.Batch

	err error
}

func newNsqClient(
	observer outputs.Observer,
	nsqd string,
	index string,
	// key *fmtstr.EventFormatString,
	// topic outil.Selector,
	topic string,
	writer codec.Codec,
	// cfg *sarama.Config,
	writeTimeout time.Duration,
	dialTimeout time.Duration,
) (*client, error) {
	cfg := nsq.NewConfig()
	cfg.WriteTimeout = writeTimeout
	cfg.DialTimeout = dialTimeout
	c := &client{
		log:      logp.NewLogger(logSelector),
		observer: observer,
		nsqd:     nsqd,
		topic:    topic,
		// key:      key,
		index:  strings.ToLower(index),
		codec:  writer,
		config: cfg,
	}

	return c, nil
}

func (c *client) Connect() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.log.Debugf("connect: %v", c.nsqd)

	// try to connect
	// producer, err := sarama.NewAsyncProducer(c.nsqd, &c.config)
	// if err != nil {
	// 	c.log.Errorf("nsq connect fails with: %+v", err)
	// 	return err
	// }
	producer, err := nsq.NewProducer(c.nsqd, c.config)
	if err != nil {
		logp.Err("[main:NsqForward.Open] NewProducer error ", err)
		c.log.Errorf("nsq connect fails with: %+v", err)
		return err
	}

	// todo: set logger
	// pruducer.SetLogger(c.log, LogLevelInfo)
	c.producer = producer

	// c.wg.Add(2)
	// go c.successWorker(producer.Successes())
	// go c.errorWorker(producer.Errors())

	return nil
}

func (c *client) Publish(_ context.Context, batch publisher.Batch) error {
	events := batch.Events()
	c.observer.NewBatch(len(events))

	// ref := &msgRef{
	// 	client: c,
	// 	count:  int32(len(events)),
	// 	total:  len(events),
	// 	failed: nil,
	// 	batch:  batch,
	// }

	st := c.observer

	msgs, err := c.buildNsqMessages(events)
	dropped := len(events) - len(msgs)
	// c.log.Info("events=%v msgs=%v", len(events), len(msgs))
	if err != nil {
		c.log.Errorf("[main:nsq] c.buildNsqMessages %v", err)
		c.observer.Failed(len(events))
		batch.RetryEvents(events)
		return nil
	}

	// nsq send failed do retry...
	err = c.producer.MultiPublish(c.topic, msgs)
	if err != nil {
		c.observer.Failed(len(events))
		batch.RetryEvents(events)
		return err
	}
	batch.ACK()

	st.Dropped(dropped)
	st.Acked(len(msgs))
	return err
}

func (c *client) buildNsqMessages(events []publisher.Event) ([][]byte, error) {
	length := len(events)
	msgs := make([][]byte, length)
	var count int
	var err error

	for idx := 0; idx < length; idx++ {
		event := events[idx].Content
		serializedEvent, nerr := c.codec.Encode(c.index, &event)
		if nerr != nil {
			c.log.Errorf("[main:nsq] c.codec.Encode fail %v", err)
			err = nerr
		} else {
			tmp := string(serializedEvent)
			msgs[count] = []byte(tmp)
			count++
		}
	}

	return msgs[:count], err
}

// type publishFn func(
// 	keys outil.Selector,
// 	data []publisher.Event,
// ) ([]publisher.Event, error)

// type client struct {
// 	outputs.NetworkClient

// }
