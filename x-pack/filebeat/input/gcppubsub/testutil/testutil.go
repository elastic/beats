// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package testutil

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

	"github.com/elastic/beats/v7/libbeat/tests/compose"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/iterator"
)

const (
	emulatorProjectID = "test-project-id"
	emulatorTopic     = "test-topic-foo"
)

var once sync.Once

func TestSetup(t *testing.T) (*pubsub.Client, context.CancelFunc) {
	t.Helper()

	var host string
	if IsInDockerIntegTestEnv() {
		// We're running inside of integration test environment so
		// make sure that that googlepubsub container is running.
		host = compose.EnsureUp(t, "googlepubsub").Host()
		os.Setenv("PUBSUB_EMULATOR_HOST", host)
	} else {
		host = os.Getenv("PUBSUB_EMULATOR_HOST")
		if host == "" {
			t.Skip("PUBSUB_EMULATOR_HOST is not set in environment. You can start " +
				"the emulator with \"docker compose up\" from the _meta directory. " +
				"The default address is PUBSUB_EMULATOR_HOST=localhost:8432")
		}
	}

	once.Do(func() {

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

func CreateTopic(t *testing.T, client *pubsub.Client) {
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

func CreateSubscription(t *testing.T, subscription string, client *pubsub.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := client.Subscription(subscription)
	exists, err := sub.Exists(ctx)
	if err != nil {
		t.Fatalf("failed to check if sub exists: %v", err)
	}
	if exists {
		return
	}

	sub, err = client.CreateSubscription(ctx, subscription, pubsub.SubscriptionConfig{
		Topic: client.Topic(emulatorTopic),
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	t.Log("New subscription created:", sub.ID())
}

func PublishMessages(t *testing.T, client *pubsub.Client, numMsgs int) []string {
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

func IsInDockerIntegTestEnv() bool {
	return os.Getenv("BEATS_INSIDE_INTEGRATION_TEST_ENV") != ""
}
