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

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "kafka")

	c, err := startConsumer(t, "metricbeat-test")
	if err != nil {
		t.Fatal(errors.Wrap(err, "starting kafka consumer"))
	}
	defer c.Close()

	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	err = mbtest.WriteEventsReporterV2(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func startConsumer(t *testing.T, topic string) (io.Closer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V0_10_2_1
	client, err := sarama.NewClient([]string{getTestKafkaHost()}, config)
	if err != nil {
		t.Errorf("%s", err)
		t.FailNow()
	}

	groupID := "test-group"
	broker, err := client.Coordinator(groupID)
	if err != nil {
		return nil, errors.Wrap(err, "getting coordinator")
	}

	resp, err := broker.JoinGroup(&sarama.JoinGroupRequest{
		GroupId:        groupID,
		ProtocolType:   "consumer",
		SessionTimeout: 30000, // milliseconds
	})
	if err != nil {
		return nil, errors.Wrap(err, "joining consumer group")
	}
	if resp.Err != sarama.ErrNoError {
		return nil, errors.Wrap(resp.Err, "joining consumer group, response error")
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, errors.Wrap(err, "new consumer")
	}

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		consumer.Close()
		return nil, err
	}

	_, err = consumer.ConsumePartition(topic, partitions[0], sarama.OffsetNewest)
	if err != nil {
		consumer.Close()
		return nil, err
	}

	//m := <-c.Messages()
	//t.Log(string(m.Value))

	return consumer, nil
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"consumergroup"},
		"hosts":      []string{getTestKafkaHost()},
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
