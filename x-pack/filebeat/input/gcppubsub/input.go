// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcppubsub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/version"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/useragent"
)

const (
	inputName    = "gcp-pubsub"
	oldInputName = "google-pubsub"

	// retryInterval is the minimum duration between pub/sub client retries.
	retryInterval = 30 * time.Second
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(fmt.Errorf("failed to register %v input: %w", inputName, err))
	}

	err = input.Register(oldInputName, NewInput)
	if err != nil {
		panic(fmt.Errorf("failed to register %v input: %w", oldInputName, err))
	}
}

func configID(config *conf.C) (string, error) {
	var tmp struct {
		ID string `config:"id"`
	}
	if err := config.Unpack(&tmp); err != nil {
		return "", fmt.Errorf("error extracting ID: %w", err)
	}
	if tmp.ID != "" {
		return tmp.ID, nil
	}

	var h map[string]interface{}
	_ = config.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute ID from configuration: %w", err)
	}

	return fmt.Sprintf("%16X", id), nil
}

type pubsubInput struct {
	config

	log      *logp.Logger
	outlet   channel.Outleter // Output of received pubsub messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerSem *awscommon.Sem

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on pubsub worker goroutine.

	id      string // id is the ID for metrics registration.
	metrics *inputMetrics
}

// NewInput creates a new Google Cloud Pub/Sub input that consumes events from
// a topic subscription.
func NewInput(cfg *conf.C, connector channel.Connector, inputContext input.Context) (inp input.Input, err error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err = cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	id, err := configID(cfg)
	if err != nil {
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
		workerSem:    awscommon.NewSem(conf.Subscription.NumGoroutines),
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		id:           id,
	}

	// Build outlet for events.
	in.outlet, err = connector.ConnectWith(cfg, beat.ClientConfig{
		EventListener: acker.ConnectionOnly(
			acker.EventPrivateReporter(func(_ int, privates []interface{}) {
				for _, priv := range privates {
					if msg, ok := priv.(*pubsub.Message); ok {
						msg.Ack()

						in.metrics.ackedMessageCount.Inc()
						in.metrics.bytesProcessedTotal.Add(uint64(len(msg.Data)))
						in.metrics.processingTime.Update(time.Since(msg.PublishTime).Nanoseconds())
					} else {
						in.metrics.failedAckedMessageCount.Inc()
						in.log.Error("Failed ACKing pub/sub event")
					}
				}
			}),
		),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: boolPtr(false),
		},
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
		in.metrics = newInputMetrics(in.id, nil)
		in.workerWg.Add(1)
		go func() {
			in.log.Info("Pub/Sub input worker has started.")
			defer in.log.Info("Pub/Sub input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()

			// Throttle pubsub client restarts.
			rt := rate.NewLimiter(rate.Every(retryInterval), 1)

			// Watchdog to keep the worker operating after an error.
			for in.workerCtx.Err() == nil {
				// Rate limit.
				if err := rt.Wait(in.workerCtx); err != nil {
					continue
				}

				if err := in.run(); err != nil {
					if in.workerCtx.Err() == nil {
						in.log.Warnw("Restarting failed Pub/Sub input worker.", "error", err)
						continue
					}

					// Log any non-cancellation error before stopping.
					if !errors.Is(err, context.Canceled) {
						in.log.Errorw("Pub/Sub input worker failed.", "error", err)
					}
				}
			}
		}()
	})
}

func (in *pubsubInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	numGoRoutines := in.Subscription.NumGoroutines
	var workerWg sync.WaitGroup

	for ctx.Err() == nil {
		workers, err := in.workerSem.AcquireContext(numGoRoutines, ctx)
		if err != nil {
			break
		}

		if workers == 0 {
			continue
		}
		workerWg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				client, err := in.newPubsubClient(ctx)
				defer func() {
					in.workerSem.Release(1)
					workerWg.Done()
				}()
				if err != nil {
					in.log.Errorw("failed to create pub/sub client: ", "error", err)
					cancel()
				} else {
					defer client.Close()
				}

				// Setup our subscription to the topic.
				sub, err := in.getOrCreateSubscription(ctx, client)
				if err != nil {
					in.log.Error("failed to subscribe to pub/sub topic: ", err)
					cancel()
				}
				sub.ReceiveSettings.NumGoroutines = in.Subscription.NumGoroutines
				sub.ReceiveSettings.MaxOutstandingMessages = in.Subscription.MaxOutstandingMessages

				// Start receiving messages.
				topicID := makeTopicID(in.ProjectID, in.Topic)
				receiveErr := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
					if ok := in.outlet.OnEvent(makeEvent(topicID, msg)); !ok {
						msg.Nack()
						in.metrics.nackedMessageCount.Inc()
						in.log.Debug("OnEvent returned false. Stopping input worker.")
						cancel()
					}
				})
				if receiveErr != nil {
					in.log.Error("Failed receiving pub/sub event.", "error", receiveErr)
				}
			}()
		}
	}

	// Wait for all workers to finish.
	workerWg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}

// Stop stops the pubsub input and waits for it to fully stop.
func (in *pubsubInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
	in.metrics.Close()
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
		event.Fields["labels"] = msg.Attributes
	}

	return event
}

func (in *pubsubInput) getOrCreateSubscription(ctx context.Context, client *pubsub.Client) (*pubsub.Subscription, error) {
	sub := client.Subscription(in.Subscription.Name)

	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check if subscription exists: %w", err)
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
			return nil, fmt.Errorf("failed to create subscription: %w", err)
		}
		in.log.Debug("Created new subscription.")
		return sub, nil
	}

	return nil, errors.New("no subscription exists and 'subscription.create' is not enabled")
}

func (in *pubsubInput) newPubsubClient(ctx context.Context) (*pubsub.Client, error) {
	opts := []option.ClientOption{option.WithUserAgent(useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String()))}

	if in.AlternativeHost != "" {
		// This will be typically set because we want to point the input to a testing pubsub emulator.
		conn, err := grpc.Dial(in.AlternativeHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
