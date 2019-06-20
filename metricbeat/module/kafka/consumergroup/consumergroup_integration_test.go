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

// +build integration

package consumergroup

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	saramacluster "github.com/bsm/sarama-cluster"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/tests/compose"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"

	kafkaSASLConsumerUsername = "consumer"
	kafkaSASLConsumerPassword = "consumer-secret"
	kafkaSASLUsername         = "stats"
	kafkaSASLPassword         = "test-secret"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "kafka")

	c, err := startConsumer(t, "metricbeat-test")
	if err != nil {
		t.Fatal(errors.Wrap(err, "starting kafka consumer"))
	}
	defer c.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	for retries := 0; retries < 3; retries++ {
		err = mbtest.WriteEventsReporterV2Error(ms, t, "")
		if err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("write", err)
}

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "kafka")

	c, err := startConsumer(t, "metricbeat-test")
	if err != nil {
		t.Fatal(errors.Wrap(err, "starting kafka consumer"))
	}
	defer c.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	var data []mb.Event
	var errors []error
	for retries := 0; retries < 3; retries++ {
		data, errors = mbtest.ReportingFetchV2Error(f)
		if len(data) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if len(errors) > 0 {
		t.Fatalf("fetch %v", errors)
	}
	if len(data) == 0 {
		t.Fatalf("No consumer groups fetched")
	}
}

func startConsumer(t *testing.T, topic string) (io.Closer, error) {
	brokers := []string{getTestKafkaHost()}
	topics := []string{topic}
	config := saramacluster.NewConfig()
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaSASLConsumerUsername
	config.Net.SASL.Password = kafkaSASLConsumerPassword
	return saramacluster.NewConsumer(brokers, "test-group", topics, config)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"consumergroup"},
		"hosts":      []string{getTestKafkaHost()},
		"username":   kafkaSASLUsername,
		"password":   kafkaSASLPassword,
	}
}

func getTestKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_PORT", kafkaDefaultPort),
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
