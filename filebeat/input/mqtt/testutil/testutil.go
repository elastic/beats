// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultHost = "localhost"
	defaultPort = "1883"

	waitTimeout = 30 * time.Second
)

// HostPort returns the MQTT broker URL used by integration tests.
func HostPort() string {
	return fmt.Sprintf("tcp://%s:%s",
		getOrDefault(os.Getenv("MOSQUITTO_HOST"), defaultHost), //nolint:misspell // required by env var name
		getOrDefault(os.Getenv("MOSQUITTO_PORT"), defaultPort), //nolint:misspell // required by env var name
	)
}

// CreatePublisher creates an MQTT client connected to the test broker.
func CreatePublisher(t *testing.T, clientID string) libmqtt.Client {
	t.Helper()

	clientOptions := libmqtt.NewClientOptions().
		SetClientID(clientID).
		SetAutoReconnect(false).
		SetConnectRetry(false).
		AddBroker(HostPort())
	client := libmqtt.NewClient(clientOptions)
	token := client.Connect()
	require.True(t, token.WaitTimeout(waitTimeout), "timed out connecting MQTT publisher %q", clientID)
	require.NoError(t, token.Error(), "failed to connect MQTT publisher %q", clientID)
	t.Cleanup(func() {
		client.Disconnect(250)
	})
	return client
}

// PublishMessage publishes a single message to the given topic with QoS 1.
func PublishMessage(t *testing.T, publisher libmqtt.Client, topic, message string) {
	t.Helper()

	token := publisher.Publish(topic, 1, false, []byte(message))
	require.True(t, token.WaitTimeout(waitTimeout), "timed out publishing to topic %q", topic)
	require.NoError(t, token.Error(), "failed to publish to topic %q", topic)
}

// EmitMessages periodically publishes messages until the context is cancelled.
func EmitMessages(t *testing.T, ctx context.Context, publisher libmqtt.Client, topic, message string) {
	t.Helper()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				token := publisher.Publish(topic, 1, false, []byte(message))
				if !token.WaitTimeout(waitTimeout) {
					require.Fail(t, "timed out publishing to topic %q", topic)
				}
				assert.NoError(t, token.Error(), "failed to publish to topic %q", topic)
			}
		}
	}()
}

func getOrDefault(s, defaultString string) string {
	if s == "" {
		return defaultString
	}
	return s
}
