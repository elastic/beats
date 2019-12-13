// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package googlepubsub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"google.golang.org/api/option"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/common/useragent"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	inputName = "google-pubsub"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

type pubsubInput struct {
	config

	log      *logp.Logger
	outlet   channel.Outleter // Output of received pubsub messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on pubsub worker goroutine.

	ackedCount *atomic.Uint32 // Total number of successfully ACKed pubsub messages.
}

// NewInput creates a new Google Cloud Pub/Sub input that consumes events from
// a topic subscription.
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (inp input.Input, err error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err = cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	// Wrap input.Context's Done channel with a context.Context. This goroutine
	// stops with the parent closes the Done channel.
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Done:
		case <-inputCtx.Done():
		}
	}()

	// If the input ever needs to be made restartable, then context would need
	// to be recreated with each restart.
	workerCtx, workerCancel := context.WithCancel(inputCtx)

	in := &pubsubInput{
		config: conf,
		log: logp.NewLogger("google.pubsub").With(
			"pubsub_project", conf.ProjectID,
			"pubsub_topic", conf.Topic,
			"pubsub_subscription", conf.Subscription),
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		ackedCount:   atomic.NewUint32(0),
	}

	// Build outlet for events.
	in.outlet, err = connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
		ACKEvents: func(privates []interface{}) {
			for _, priv := range privates {
				if msg, ok := priv.(*pubsub.Message); ok {
					msg.Ack()
					in.ackedCount.Inc()
				} else {
					in.log.Error("Failed ACKing pub/sub event")
				}
			}
		},
	})
	if err != nil {
		return nil, err
	}
	in.log.Info("Initialized Google Pub/Sub input.")
	return in, nil
}

// Run starts the pubsub input worker then returns. Only the first invocation
// will ever start the pubsub worker.
func (in *pubsubInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("Pub/Sub input worker has started.")
			defer in.log.Info("Pub/Sub input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

func (in *pubsubInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	// Make pubsub client.
	opts := []option.ClientOption{option.WithUserAgent(useragent.UserAgent("Filebeat"))}
	if in.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(in.CredentialsFile))
	} else if len(in.CredentialsJSON) > 0 {
		option.WithCredentialsJSON(in.CredentialsJSON)
	}

	client, err := pubsub.NewClient(ctx, in.ProjectID, opts...)
	if err != nil {
		return err
	}
	defer client.Close()

	// Setup our subscription to the topic.
	sub, err := in.getOrCreateSubscription(ctx, client)
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to pub/sub topic")
	}
	sub.ReceiveSettings.NumGoroutines = in.Subscription.NumGoroutines
	sub.ReceiveSettings.MaxOutstandingMessages = in.Subscription.MaxOutstandingMessages

	// Start receiving messages.
	topicID := makeTopicID(in.ProjectID, in.Topic)
	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if ok := in.outlet.OnEvent(makeEvent(topicID, msg)); !ok {
			msg.Nack()
			in.log.Debug("OnEvent returned false. Stopping input worker.")
			cancel()
		}
	})
}

// Stop stops the pubsub input and waits for it to fully stop.
func (in *pubsubInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *pubsubInput) Wait() {
	in.Stop()
}

// makeTopicID returns a short sha256 hash of the project ID plus topic name.
// This string can be joined with pub/sub message IDs that are unique within a
// topic to create a unique _id for documents.
func makeTopicID(project, topic string) string {
	h := sha256.New()
	h.Write([]byte(project))
	h.Write([]byte(topic))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:10]
}

func makeEvent(topicID string, msg *pubsub.Message) beat.Event {
	id := topicID + "-" + msg.ID

	fields := common.MapStr{
		"event": common.MapStr{
			"id":      id,
			"created": time.Now().UTC(),
		},
		"message": string(msg.Data),
	}
	if len(msg.Attributes) > 0 {
		fields.Put("labels", msg.Attributes)
	}

	return beat.Event{
		Timestamp: msg.PublishTime.UTC(),
		Meta: common.MapStr{
			"id": id,
		},
		Fields:  fields,
		Private: msg,
	}
}

func (in *pubsubInput) getOrCreateSubscription(ctx context.Context, client *pubsub.Client) (*pubsub.Subscription, error) {
	sub := client.Subscription(in.Subscription.Name)

	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if subscription exists")
	}
	if exists {
		return sub, nil
	}

	// Create subscription.
	if in.Subscription.Create {
		sub, err = client.CreateSubscription(ctx, in.Subscription.Name, pubsub.SubscriptionConfig{
			Topic: client.Topic(in.Topic),
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create subscription")
		}
		in.log.Debug("Created new subscription.")
		return sub, nil
	}

	return nil, errors.New("no subscription exists and 'subscription.create' is not enabled")
}
