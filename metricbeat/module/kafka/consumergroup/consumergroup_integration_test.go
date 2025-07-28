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

package consumergroup

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/sarama"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

const (
	kafkaSASLConsumerUsername = "consumer"
	kafkaSASLConsumerPassword = "consumer-secret"
	kafkaSASLUsername         = "stats"
	kafkaSASLPassword         = "test-secret"
)

type ConsumerGroupHandler struct {
	t *testing.T
}

func (ConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (ConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h *ConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	count := 0
	for msg := range claim.Messages() {
		if count > 5 {
			// just for testing, do not consume too many messages
			break
		}
		h.t.Logf("Message topic:%q partition:%d offset:%d value:%s\n",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
		// mark the message is consumed
		sess.MarkMessage(msg, "")
		count += 1
	}
	return nil
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)
	host := service.HostForPort(9092)

	c, err := startConsumer(t, host, "test-group")
	if err != nil {
		t.Fatal(fmt.Errorf("starting kafka consumer: %w", err))
	}
	defer c.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(host))
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
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)

	c, err := startConsumer(t, service.HostForPort(9092), "test-group")
	if err != nil {
		t.Fatal(fmt.Errorf("starting kafka consumer: %w", err))
	}
	defer c.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.HostForPort(9092)))

	var data []mb.Event
	var errors []error
	for retries := 0; retries < 3; retries++ {
		data, errors = mbtest.ReportingFetchV2Error(f)
		if len(data) > 0 {
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}
	if len(errors) > 0 {
		t.Fatalf("fetch %v", errors)
	}
	if len(data) == 0 {
		t.Fatalf("No consumer groups fetched")
	}

	for _, v := range data {
		if _, ok := v.MetricSetFields["consumer_lag"]; ok {
			t.Fatalf("shouldn't have consumer_lag, the consumergroup doesn't consume anything")
		}
	}
}

func TestFetchWithConsumerLag(t *testing.T) {
	groupName := "test-group-consumed"
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)
	c, err := startConsumer(t, service.HostForPort(9092), groupName)
	if err != nil {
		t.Fatal(fmt.Errorf("starting kafka consumer: %w", err))
	}
	defer c.Close()

	// consue some data
	handler := ConsumerGroupHandler{
		t: t,
	}
	c.Consume(t.Context(), []string{"test"}, &handler)

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.HostForPort(9092)))

	var data []mb.Event
	var errors []error
	for retries := 0; retries < 3; retries++ {
		data, errors = mbtest.ReportingFetchV2Error(f)
		if len(data) > 0 {
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}
	if len(errors) > 0 {
		t.Fatalf("fetch %v", errors)
	}
	if len(data) == 0 {
		t.Fatalf("No consumer groups fetched")
	}

	has_lag := false
	for _, v := range data {
		// some data won't has consumer_lag
		if _, ok := v.MetricSetFields["consumer_lag"]; ok {
			has_lag = true
		}
	}
	if !has_lag {
		t.Fatalf("consumed some messages but didn't fetch the consumer_lag metrics")
	}
}

func startConsumer(t *testing.T, host string, groupID string) (sarama.ConsumerGroup, error) {
	brokers := []string{host}

	config := sarama.NewConfig()
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaSASLConsumerUsername
	config.Net.SASL.Password = kafkaSASLConsumerPassword

	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

	// Create a new consumer group
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		t.Fatalf("Error creating consumer group: %v, brokers: %s", err, brokers)
		return nil, err
	}

	return consumerGroup, nil
}

func consumeSomeData(ctx context.Context, t *testing.T, consumerGroup sarama.ConsumerGroup, topics []string) {
	handler := ConsumerGroupHandler{
		t: t,
	}
	consumerGroup.Consume(ctx, topics, &handler)
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"consumergroup"},
		"hosts":      []string{host},
		"username":   kafkaSASLUsername,
		"password":   kafkaSASLPassword,
	}
}
