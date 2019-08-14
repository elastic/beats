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

package partition

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

const (
	kafkaSASLProducerUsername = "producer"
	kafkaSASLProducerPassword = "producer-secret"
	kafkaSASLUsername         = "stats"
	kafkaSASLPassword         = "test-secret"
)

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(120*time.Second),
		compose.UpWithAdvertisedHostEnvFile,
	)

	generateKafkaData(t, service.Host(), "metricbeat-generate-data")

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host(), ""))
	err := mbtest.WriteEventsReporterV2Error(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestTopic(t *testing.T) {
	service := compose.EnsureUp(t, "kafka")

	logp.TestingSetup(logp.WithSelectors("kafka"))

	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("test-metricbeat-%s", id)

	// Create initial topic
	generateKafkaData(t, service.Host(), testTopic)

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host(), testTopic))
	dataBefore, err := mbtest.ReportingFetchV2Error(f)
	if err != nil {
		t.Fatal("write", err)
	}
	if len(dataBefore) == 0 {
		t.Fatalf("No offsets fetched from topic (before): %v", testTopic)
	}
	t.Logf("before: %v", dataBefore)

	var n int64 = 10
	var i int64 = 0
	// Create n messages
	for ; i < n; i++ {
		generateKafkaData(t, service.Host(), testTopic)
	}

	dataAfter, err := mbtest.ReportingFetchV2Error(f)
	if err != nil {
		t.Fatal("write", err)
	}
	if len(dataAfter) == 0 {
		t.Fatalf("No offsets fetched from topic (after): %v", testTopic)
	}
	t.Logf("after: %v", dataAfter)

	// Checks that no new topics / partitions were added
	assert.True(t, len(dataBefore) == len(dataAfter))

	var offsetBefore int64 = 0
	var offsetAfter int64 = 0

	// Its possible that other topics exists -> select the right data
	for _, data := range dataBefore {
		if data.ModuleFields["topic"].(common.MapStr)["name"] == testTopic {
			offsetBefore = data.MetricSetFields["offset"].(common.MapStr)["newest"].(int64)
		}
	}

	for _, data := range dataAfter {
		if data.ModuleFields["topic"].(common.MapStr)["name"] == testTopic {
			offsetAfter = data.MetricSetFields["offset"].(common.MapStr)["newest"].(int64)
		}
	}

	// Compares offset before and after
	if offsetBefore+n != offsetAfter {
		t.Errorf("Offset before: %v", offsetBefore)
		t.Errorf("Offset after: %v", offsetAfter)
	}
	assert.True(t, offsetBefore+n == offsetAfter)
}

func generateKafkaData(t *testing.T, host string, topic string) {
	t.Logf("Send Kafka Event to topic: %v", topic)

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	// Retry for 10 seconds
	config.Producer.Retry.Max = 20
	config.Producer.Retry.Backoff = 500 * time.Millisecond
	config.Metadata.Retry.Max = 20
	config.Metadata.Retry.Backoff = 500 * time.Millisecond
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaSASLProducerUsername
	config.Net.SASL.Password = kafkaSASLProducerPassword
	client, err := sarama.NewClient([]string{host}, config)
	if err != nil {
		t.Errorf("%s", err)
		t.FailNow()
	}

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		t.Error(err)
	}
	defer producer.Close()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder("Hello World"),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		t.Errorf("failed to send message: %s\n", err)
	}

	err = client.RefreshMetadata(topic)
	if err != nil {
		t.Errorf("failed to refresh metadata for topic '%s': %s\n", topic, err)
	}
}

func getConfig(host string, topic string) map[string]interface{} {
	var topics []string
	if topic != "" {
		topics = []string{topic}
	}

	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"partition"},
		"hosts":      []string{host},
		"topics":     topics,
		"username":   kafkaSASLUsername,
		"password":   kafkaSASLPassword,
	}
}
