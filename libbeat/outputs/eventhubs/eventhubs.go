package eventhubs

import (
	"context"
	"time"

	"github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/Azure/azure-amqp-common-go/sas"

	"github.com/Azure/azure-event-hubs-go"
)

type amqpClient struct {
	codec     codec.Codec
	index     string
	observer  outputs.Observer
	namespace string
	hub       string
	keyName   string
	key       string
	client    *eventhub.Hub
}

func init() {
	outputs.RegisterType("eventhubs", makeEventHubClient)
}

func makeEventHubClient(beat beat.Info, observer outputs.Observer, cfg *common.Config) (outputs.Group, error) {
	config := defaultConfig
	err := cfg.Unpack(&config)
	if err != nil {
		return outputs.Fail(err)
	}

	var enc codec.Codec
	if config.Codec.Namespace.IsSet() {
		enc, err = codec.CreateEncoder(beat, config.Codec)
		if err != nil {
			return outputs.Fail(err)
		}
	} else {
		enc = json.New(config.Pretty, beat.Version)
	}

	index := beat.Beat

	tokenProvider, err := sas.NewTokenProvider(sas.TokenProviderWithNamespaceAndKey(config.Namespace, config.KeyName, config.Key))
	if err != nil {
		return outputs.Fail(err)
	}

	client, err := eventhub.NewHub(config.Namespace, config.Hub, tokenProvider)
	if err != nil {
		return outputs.Fail(err)
	}

	a := &amqpClient{
		index:     index,
		observer:  observer,
		codec:     enc,
		namespace: config.Namespace,
		hub:       config.Hub,
		keyName:   config.KeyName,
		key:       config.Key,
		client:    client,
	}

	return outputs.Success(1, 5, a)
}

func (a *amqpClient) Close() error {
	if a.client != nil {
		return a.client.Close()
	} else {
		return nil
	}
}

func (a *amqpClient) Publish(batch publisher.Batch) error {
	st := a.observer
	events := batch.Events()

	st.NewBatch(len(events))

	if len(events) == 0 {
		batch.ACK()
		return nil
	}

	for i := range events {
		err := a.publishEvent(&events[i])
		if err != nil {
			events = events[i:]

			// Return events to pipeline to be retried
			batch.RetryEvents(events)
			logp.Err("Failed to publish events caused by: %v", err)

			// Record Stats
			st.Acked(i)
			st.Failed(len(events))
			return err
		}
	}

	// Ack that the batch has been sent
	batch.ACK()

	// Record stats
	st.Acked(len(events))
	return nil
}

func (a *amqpClient) publishEvent(event *publisher.Event) error {
	serializedEvent, err := a.codec.Encode(a.index, &event.Content)
	if err != nil {
		if !event.Guaranteed() {
			return err
		}
		logp.Critical("Unable to encode event: %v", err)
		return err
	}

	messageId, err := uuid.NewV4()
	if err != nil {
		if !event.Guaranteed() {
			return err
		}
		logp.Critical("Unable to create new UUID: %v", err)
		return err
	}

	msg := &eventhub.Event{
		Data: serializedEvent,
		Properties: map[string]interface{}{
			"Type":      "Beats",
			"Beat":      a.index,
			"MessageId": messageId.String(),
			"Timestamp": event.Content.Timestamp.Format("2006-01-02T15:04:05.000+00:00"),
		},
		ID: messageId.String(),
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err = a.client.Send(ctx, msg)
	if err != nil {
		logp.Critical("Unable to send event: %v", err)
		return err
	}
	return nil
}
