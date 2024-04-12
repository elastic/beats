// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcppubsub

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	emulatorProjectID    = "test-project-id"
	emulatorTopic        = "test-topic-foo"
	emulatorSubscription = "test-subscription-bar"
)

var once sync.Once

func testSetup(t *testing.T) (*pubsub.Client, context.CancelFunc) {
	t.Helper()

	var host string
	if isInDockerIntegTestEnv() {
		// We're running inside out integration test environment so
		// make sure that that googlepubsub container is running.
		host = compose.EnsureUp(t, "googlepubsub").Host()
		os.Setenv("PUBSUB_EMULATOR_HOST", host)
	} else {
		host = os.Getenv("PUBSUB_EMULATOR_HOST")
		if host == "" {
			t.Skip("PUBSUB_EMULATOR_HOST is not set in environment. You can start " +
				"the emulator with \"docker-compose up\" from the _meta directory. " +
				"The default address is PUBSUB_EMULATOR_HOST=localhost:8432")
		}
	}

	once.Do(func() {
		logp.TestingSetup()

		// Disable HTTP keep-alives to ensure no extra goroutines hang around.
		httpClient := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

		// Sanity check the emulator.
		//nolint:noctx // this is just for the tests
		resp, err := httpClient.Get("http://" + host)
		if err != nil {
			t.Fatalf("pubsub emulator at %s is not healthy: %v", host, err)
		}
		defer resp.Body.Close()

		_, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal("failed to read response", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("pubsub emulator is not healthy, got status code %d", resp.StatusCode)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, emulatorProjectID)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resetPubSub(t, client)
	return client, cancel
}

func resetPubSub(t *testing.T, client *pubsub.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clear topics.
	topics := client.Topics(ctx)
	for {
		topic, err := topics.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = topic.Delete(ctx); err != nil {
			t.Fatalf("failed to delete topic %v: %v", topic.ID(), err)
		}
	}

	// Clear subscriptions.
	subs := client.Subscriptions(ctx)
	for {
		sub, err := subs.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if err = sub.Delete(ctx); err != nil {
			t.Fatalf("failed to delete subscription %v: %v", sub.ID(), err)
		}
	}
}

func createTopic(t *testing.T, client *pubsub.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := client.Topic(emulatorTopic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		t.Fatalf("failed to check if topic exists: %v", err)
	}
	if !exists {
		if topic, err = client.CreateTopic(ctx, emulatorTopic); err != nil {
			t.Fatalf("failed to create the topic: %v", err)
		}
		t.Log("Topic created:", topic.ID())
	}
}

func publishMessages(t *testing.T, client *pubsub.Client, numMsgs int) []string {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := client.Topic(emulatorTopic)
	defer topic.Stop()

	messageIDs := make([]string, numMsgs)
	for i := 0; i < numMsgs; i++ {
		result := topic.Publish(ctx, &pubsub.Message{
			Data: []byte(time.Now().UTC().Format(time.RFC3339Nano) + ": hello world " + strconv.Itoa(i)),
		})

		// Wait for message to publish and get assigned ID.
		id, err := result.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		messageIDs[i] = id
	}
	t.Logf("Published %d messages to topic %v. ID range: [%v, %v]", len(messageIDs), topic.ID(), messageIDs[0], messageIDs[len(messageIDs)-1])
	return messageIDs
}

func createSubscription(t *testing.T, client *pubsub.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := client.Subscription(emulatorSubscription)
	exists, err := sub.Exists(ctx)
	if err != nil {
		t.Fatalf("failed to check if sub exists: %v", err)
	}
	if exists {
		return
	}

	sub, err = client.CreateSubscription(ctx, emulatorSubscription, pubsub.SubscriptionConfig{
		Topic: client.Topic(emulatorTopic),
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	t.Log("New subscription created:", sub.ID())
}

func ifNotDone(ctx context.Context, f func()) func() {
	return func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		f()
	}
}

func defaultTestConfig() *conf.C {
	return conf.MustNewConfigFrom(map[string]interface{}{
		"project_id": emulatorProjectID,
		"topic":      emulatorTopic,
		"subscription": map[string]interface{}{
			"name":   emulatorSubscription,
			"create": true,
		},
		"credentials_file": "testdata/fake.json",
	})
}

func isInDockerIntegTestEnv() bool {
	return os.Getenv("BEATS_INSIDE_INTEGRATION_TEST_ENV") != ""
}

func runTest(t *testing.T, cfg *conf.C, run func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T)) {
	runTestWithACKer(t, cfg, ackEvent, run)
}

func runTestWithACKer(t *testing.T, cfg *conf.C, onEvent eventHandler, run func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T)) {
	if !isInDockerIntegTestEnv() {
		// Don't test goroutines when using our compose.EnsureUp.
		defer resources.NewGoroutinesChecker().Check(t)
	}

	// Create pubsub client for setting up and communicating to emulator.
	client, clientCancel := testSetup(t)
	defer clientCancel()
	defer client.Close()

	// Simulate input.Context from Filebeat input runner.
	inputCtx := newInputContext()
	defer close(inputCtx.Done)

	// Stub outlet for receiving events generated by the input.
	eventOutlet := newStubOutlet(onEvent)
	defer eventOutlet.Close()

	connector := channel.ConnectorFunc(func(_ *conf.C, cliCfg beat.ClientConfig) (channel.Outleter, error) {
		eventOutlet.setClientConfig(cliCfg)
		return eventOutlet, nil
	})

	in, err := NewInput(cfg, connector, inputCtx)
	if err != nil {
		t.Fatal(err)
	}
	pubsubInput := in.(*pubsubInput)
	defer pubsubInput.Stop()

	run(client, pubsubInput, eventOutlet, t)
}

func newInputContext() input.Context {
	return input.Context{
		Done: make(chan struct{}),
	}
}

type eventHandler func(beat.Event, beat.ClientConfig) bool

type stubOutleter struct {
	sync.Mutex
	cond         *sync.Cond
	done         bool
	Events       []beat.Event
	clientCfg    beat.ClientConfig
	eventHandler eventHandler
}

func newStubOutlet(onEvent eventHandler) *stubOutleter {
	o := &stubOutleter{
		eventHandler: onEvent,
	}
	o.cond = sync.NewCond(o)
	return o
}

func ackEvent(ev beat.Event, cfg beat.ClientConfig) bool {
	if cfg.EventListener == nil {
		return false
	}

	cfg.EventListener.AddEvent(ev, true)
	cfg.EventListener.ACKEvents(1)
	return true
}

func (o *stubOutleter) setClientConfig(cfg beat.ClientConfig) {
	o.Lock()
	defer o.Unlock()
	o.clientCfg = cfg
}

func (o *stubOutleter) waitForEvents(numEvents int) ([]beat.Event, bool) {
	o.Lock()
	defer o.Unlock()

	for len(o.Events) < numEvents && !o.done {
		o.cond.Wait()
	}

	size := numEvents
	if size >= len(o.Events) {
		size = len(o.Events)
	}

	out := make([]beat.Event, size)
	copy(out, o.Events)
	return out, len(out) == numEvents
}

func (o *stubOutleter) Close() error {
	o.Lock()
	defer o.Unlock()
	o.done = true
	return nil
}

func (o *stubOutleter) Done() <-chan struct{} { return nil }

func (o *stubOutleter) OnEvent(event beat.Event) bool {
	o.Lock()
	defer o.Unlock()
	acked := o.eventHandler(event, o.clientCfg)
	if acked {
		o.Events = append(o.Events, event)
		o.cond.Broadcast()
	}
	return !o.done
}

// --- Test Cases

func TestTopicDoesNotExist(t *testing.T) {
	cfg := defaultTestConfig()

	runTest(t, cfg, func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T) {
		require.Error(t, input.run())
	})
}

func TestSubscriptionDoesNotExistError(t *testing.T) {
	cfg := defaultTestConfig()
	_ = cfg.SetBool("subscription.create", -1, false)

	runTest(t, cfg, func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T) {
		createTopic(t, client)

		err := input.run()
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "no subscription exists and 'subscription.create' is not enabled")
		}
	})
}

func TestSubscriptionExists(t *testing.T) {
	cfg := defaultTestConfig()

	runTest(t, cfg, func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T) {
		createTopic(t, client)
		createSubscription(t, client)
		publishMessages(t, client, 5)

		var group errgroup.Group
		group.Go(input.run)

		time.AfterFunc(10*time.Second, func() { out.Close() })
		events, ok := out.waitForEvents(5)
		if !ok {
			t.Fatalf("Expected 5 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSubscriptionCreate(t *testing.T) {
	cfg := defaultTestConfig()

	runTest(t, cfg, func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T) {
		createTopic(t, client)

		group, ctx := errgroup.WithContext(context.Background())
		group.Go(input.run)

		time.AfterFunc(1*time.Second, ifNotDone(ctx, func() { publishMessages(t, client, 5) }))
		time.AfterFunc(10*time.Second, func() { out.Close() })

		events, ok := out.waitForEvents(5)
		if !ok {
			t.Fatalf("Expected 5 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRunStop(t *testing.T) {
	cfg := defaultTestConfig()

	runTest(t, cfg, func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T) {
		input.Run()
		input.Stop()
		input.Run()
		input.Stop()
	})
}

func TestEndToEndACK(t *testing.T) {
	cfg := defaultTestConfig()

	var count atomic.Int
	seen := make(map[string]struct{})
	// ACK every other message
	halfAcker := func(ev beat.Event, clientConfig beat.ClientConfig) bool {
		msg := ev.Private.(*pubsub.Message)
		seen[msg.ID] = struct{}{}
		if count.Inc()&1 != 0 {
			// Nack will result in the Message being redelivered more quickly than if it were allowed to expire.
			msg.Nack()
			return false
		}
		return ackEvent(ev, clientConfig)
	}

	runTestWithACKer(t, cfg, halfAcker, func(client *pubsub.Client, input *pubsubInput, out *stubOutleter, t *testing.T) {
		createTopic(t, client)
		createSubscription(t, client)

		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		const numMsgs = 10
		publishMessages(t, client, numMsgs)
		events, ok := out.waitForEvents(numMsgs)
		if !ok {
			t.Fatalf("Expected %d events, but got %d.", 1, len(events))
		}

		// Assert that all messages were eventually received
		assert.Len(t, events, len(seen))
		got := make(map[string]struct{})
		for _, ev := range events {
			msg := ev.Private.(*pubsub.Message)
			got[msg.ID] = struct{}{}
		}
		for id := range seen {
			_, exists := got[id]
			assert.True(t, exists)
		}

		assert.EqualValues(t, input.metrics.ackedMessageCount.Get(), len(seen))

		input.Stop()
		out.Close()
		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}
