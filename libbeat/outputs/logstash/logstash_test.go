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

package logstash

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/transptest"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	v2 "github.com/elastic/go-lumber/server/v2"
)

const (
	logstashDefaultHost     = "localhost"
	logstashTestDefaultPort = "5044"
)

func TestLogstashTCP(t *testing.T) {
	enableLogging([]string{"*"})

	timeout := 2 * time.Second
	server := transptest.NewMockServerTCP(t, timeout, "", nil)

	config := map[string]interface{}{
		"hosts":   []string{server.Addr()},
		"index":   testLogstashIndex("logstash-conn-tcp"),
		"timeout": "2s",
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func TestLogstashTLS(t *testing.T) {
	enableLogging([]string{"*"})

	certName := "ca_test"

	timeout := 2 * time.Second
	transptest.GenCertForTestingPurpose(t, certName, "", "127.0.0.1", "127.0.1.1")
	server := transptest.NewMockServerTLS(t, timeout, certName, nil)

	// create lumberjack output client
	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-conn-tls"),
		"timeout":                     "2s",
		"ssl.certificate_authorities": []string{certName + ".pem"},
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func TestLogstashInvalidTLSInsecure(t *testing.T) {
	certName := "ca_invalid_test"
	ip := "1.2.3.4"

	timeout := 2 * time.Second
	transptest.GenCertForTestingPurpose(t, certName, "", ip)
	server := transptest.NewMockServerTLS(t, timeout, certName, nil)

	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-conn-tls-invalid"),
		"timeout":                     2,
		"max_retries":                 1,
		"ssl.verification_mode":       "none",
		"ssl.certificate_authorities": []string{certName + ".pem"},
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func testLogstashIndex(test string) string {
	return fmt.Sprintf("beat-logstash-int-%v-%d", test, os.Getpid())
}

func testConnectionType(
	t *testing.T,
	mock *transptest.MockServer,
	makeOutputer func() outputs.NetworkClient,
) {
	t.Log("testConnectionType")
	server, _ := v2.NewWithListener(mock.Listener)

	// worker loop
	go func() {
		defer server.Close()

		t.Log("start worker loop")
		defer t.Log("stop worker loop")

		t.Log("make outputter")
		output := makeOutputer()
		t.Logf("new outputter: %v", output)

		err := output.Connect()
		if err != nil {
			t.Error("test client failed to connect: ", err)
			return
		}

		sig := make(chan struct{})

		t.Log("publish event")
		batch := outest.NewBatch(testEvent())
		batch.OnSignal = func(_ outest.BatchSignal) {
			close(sig)
		}
		err = output.Publish(context.Background(), batch)

		t.Log("wait signal")
		<-sig

		assert.NoError(t, err)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	}()

	for batch := range server.ReceiveChan() {
		batch.ACK()

		events := batch.Events
		assert.Equal(t, 1, len(events))
		msg := events[0].(map[string]interface{})
		assert.Equal(t, 10.0, msg["extra"])
		assert.Equal(t, "message", msg["message"])
	}
}

func testEvent() beat.Event {
	return beat.Event{Fields: mapstr.M{
		"@timestamp": common.Time(time.Now()),
		"type":       "log",
		"extra":      10,
		"message":    "message",
	}}
}

func testOutputerFactory(
	t *testing.T,
	test string,
	config map[string]interface{},
) func() outputs.NetworkClient {
	return func() outputs.NetworkClient {
		return newTestLumberjackOutput(t, test, config)
	}
}

func newTestLumberjackOutput(
	t *testing.T,
	test string,
	config map[string]interface{},
) outputs.NetworkClient {
	if config == nil {
		config = map[string]interface{}{
			"hosts": []string{getLogstashHost()},
			"index": testLogstashIndex(test),
		}
	}

	cfg, _ := common.NewConfigFrom(config)
	grp, err := outputs.Load(nil, beat.Info{}, nil, "logstash", cfg)
	if err != nil {
		t.Fatalf("init logstash output plugin failed: %v", err)
	}

	client := grp.Clients[0].(outputs.NetworkClient)
	if err := client.Connect(); err != nil {
		t.Fatalf("Client failed to connected: %v", err)
	}

	return client
}

func getLogstashHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("LS_HOST", logstashDefaultHost),
		getenv("LS_TCP_PORT", logstashTestDefaultPort),
	)
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}
