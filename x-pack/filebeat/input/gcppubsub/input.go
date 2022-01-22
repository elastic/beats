// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcppubsub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/version"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/useragent"
)

const (
	inputName    = "gcp-pubsub"
	oldInputName = "google-pubsub"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}

	err = input.Register(oldInputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", oldInputName))
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
	cfg *conf.C,
	connector channel.Connector,
	inputContext input.Context,
) (inp input.Input, err error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err = cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	logger := logp.NewLogger("gcp.pubsub").With(
		"pubsub_project", conf.ProjectID,
		"pubsub_topic", conf.Topic,
		"pubsub_subscription", conf.Subscription)

	if conf.Type == oldInputName {
		logger.Warnf("%s input name is deprecated, please use %s instead", oldInputName, inputName)
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
		config:       conf,
		log:          logger,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		ackedCount:   atomic.NewUint32(0),
	}

	// Build outlet for events.
	in.outlet, err = connector.ConnectWith(cfg, beat.ClientConfig{
		ACKHandler: acker.ConnectionOnly(
			acker.EventPrivateReporter(func(_ int, privates []interface{}) {
				for _, priv := range privates {
					if msg, ok := priv.(*pubsub.Message); ok {
						msg.Ack()
						in.ackedCount.Inc()
					} else {
						in.log.Error("Failed ACKing pub/sub event")
					}
				}
			}),
		),
	})
	if err != nil {
		return nil, err
	}
	in.log.Info("Initialized GCP Pub/Sub input.")
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

	client, err := in.newPubsubClient(ctx)
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
		messages := in.parseMultipleMessages(msg.Data)
		arrayOffset := int64(0)
		for _, item := range messages {
			if in.config.Split != nil {
				split, err := newSplit(in.config.Split, in.log)
				if err != nil {
					return
				}
				// We want to be able to identify which split is the root of the chain.
				split.isRoot = true

				eventsCh, err := split.startSplit([]byte(item))
				if err != nil {
					return
				}
				for maybeMsg := range eventsCh {
					if maybeMsg.failed() {
						in.log.Errorf("error processing response: %v", maybeMsg)
						continue
					}

					// data, _ := json.Marshal(maybeMsg.msg)
					event := makeSplitEvent(topicID, msg, maybeMsg.msg, arrayOffset)
					if ok := in.outlet.OnEvent(event); !ok {
						msg.Nack()
						in.log.Debug("OnEvent returned false. Stopping input worker.")
						cancel()
					}
					arrayOffset++
				}
			} else {
				var object common.MapStr
				err = json.Unmarshal([]byte(item), &object)
				event := makeSplitEvent(topicID, msg, object, arrayOffset)
				if ok := in.outlet.OnEvent(event); !ok {
					msg.Nack()
					in.log.Debug("OnEvent returned false. Stopping input worker.")
					cancel()
				}
				arrayOffset++
			}
		}
	})
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func (in *pubsubInput) parseMultipleMessages(bMessage []byte) []string {
	var mapObject common.MapStr
	var messages []string
	// check if the message is a "records" object containing a list of events
	err := json.Unmarshal(bMessage, &mapObject)
	if err == nil {
		js, err := json.Marshal(mapObject)
		if err != nil {
			in.log.Errorw(fmt.Sprintf("serializing message %s", js), "error", err)
		}
		messages = append(messages, string(js))
	} else {
		in.log.Debugf("deserializing message into object returning error: %s", err)
		// in some cases the message is an array
		var arrayObject []common.MapStr
		err = json.Unmarshal(bMessage, &arrayObject)
		if err != nil {
			// return entire message
			in.log.Debugf("deserializing multiple messages to an array returning error: %s", err)
			messages = append(messages, string(bMessage))
		}
		in.log.Debugf("deserializing multiple messages to an array")
		for _, ms := range arrayObject {
			js, err := json.Marshal(ms)
			if err != nil {
				in.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
			messages = append(messages, string(js))
		}
	}
	return messages
}

// Stop stops the pubsub input and waits for it to fully stoin.
func (in *pubsubInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stoin.
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

func makeSplitEvent(topicID string, msg *pubsub.Message, data common.MapStr, offset int64) beat.Event {
	id := fmt.Sprintf("%s-%s-%012d", topicID, msg.ID, offset)
	message, _ := json.Marshal(data)
	event := beat.Event{
		Timestamp: msg.PublishTime.UTC(),
		Fields: common.MapStr{
			"event": common.MapStr{
				"id":      id,
				"created": time.Now().UTC(),
			},
			"message": string(message),
		},
		Private: msg,
	}
	event.SetID(id)

	if len(msg.Attributes) > 0 {
		event.PutValue("labels", msg.Attributes)
	}

	return event
}

func makeEvent(topicID string, msg *pubsub.Message) beat.Event {
	id := topicID + "-" + msg.ID

	event := beat.Event{
		Timestamp: msg.PublishTime.UTC(),
		Fields: mapstr.M{
			"event": mapstr.M{
				"id":      id,
				"created": time.Now().UTC(),
			},
			"message": string(msg.Data),
		},
		Private: msg,
	}
	event.SetID(id)

	if len(msg.Attributes) > 0 {
		event.PutValue("labels", msg.Attributes)
	}

	return event
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

func (in *pubsubInput) newPubsubClient(ctx context.Context) (*pubsub.Client, error) {
	opts := []option.ClientOption{option.WithUserAgent(useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String()))}

	if in.AlternativeHost != "" {
		// this will be typically set because we want to point the input to a testing pubsub emulator
		conn, err := grpc.Dial(in.AlternativeHost, grpc.WithInsecure())
		if err != nil {
			return nil, fmt.Errorf("cannot connect to alternative host %q: %w", in.AlternativeHost, err)
		}
		opts = append(opts, option.WithGRPCConn(conn), option.WithTelemetryDisabled())
	}

	if in.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(in.CredentialsFile))
	} else if len(in.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(in.CredentialsJSON))
	}

	return pubsub.NewClient(ctx, in.ProjectID, opts...)
}
