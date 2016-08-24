// +build integration

package kafka

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode/modetest"
	"github.com/stretchr/testify/assert"
)

const (
	kafkaDefaultHost = "localhost"
	kafkaDefaultPort = "9092"
)

func TestKafkaPublish(t *testing.T) {
	single := modetest.SingleEvent

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"kafka"})
	}

	id := strconv.Itoa(rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Int())
	testTopic := fmt.Sprintf("test-libbeat-%s", id)
	logType := fmt.Sprintf("log-type-%s", id)

	tests := []struct {
		title  string
		config map[string]interface{}
		topic  string
		events []modetest.EventInfo
	}{
		{
			"publish single event to test topic",
			nil,
			testTopic,
			single(common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
				"message":    id,
			}),
		},
		{
			"publish single event with topic from type",
			map[string]interface{}{
				"topic": "%{[type]}",
			},
			logType,
			single(common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       logType,
				"message":    id,
			}),
		},
		{
			"batch publish to test topic",
			nil,
			testTopic,
			randMulti(5, 100, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
			}),
		},
		{
			"batch publish to test topic from type",
			map[string]interface{}{
				"topic": "%{[type]}",
			},
			logType,
			randMulti(5, 100, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       logType,
			}),
		},
		{
			"batch publish with random partitioner",
			map[string]interface{}{
				"partition.random": map[string]interface{}{
					"group_events": 1,
				},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
			}),
		},
		{
			"batch publish with round robin partitioner",
			map[string]interface{}{
				"partition.round_robin": map[string]interface{}{
					"group_events": 1,
				},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
			}),
		},
		{
			"batch publish with hash partitioner without key (fallback to random)",
			map[string]interface{}{
				"partition.hash": map[string]interface{}{},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with hash partitioner with key",
			map[string]interface{}{
				"key":            "%{[message]}",
				"partition.hash": map[string]interface{}{},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
			}),
		},
		{
			// warning: this test uses random keys. In case keys are reused, test might fail.
			"batch publish with fields hash partitioner",
			map[string]interface{}{
				"partition.hash.hash": []string{
					"@timestamp",
					"type",
					"message",
				},
			},
			testTopic,
			randMulti(1, 10, common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
			}),
		},
	}

	defaultConfig := map[string]interface{}{
		"hosts":   []string{getTestKafkaHost()},
		"topic":   testTopic,
		"timeout": "1s",
	}

	for i, test := range tests {
		t.Logf("run test(%v): %v", i, test.title)

		cfg := makeConfig(t, defaultConfig)
		if test.config != nil {
			cfg.Merge(makeConfig(t, test.config))
		}

		// create output within function scope to guarantee
		// output is properly closed between single tests
		func() {
			tmp, err := New("libbeat", cfg, 0)
			if err != nil {
				t.Fatal(err)
			}

			output := tmp.(*kafka)
			defer output.Close()

			// publish test events
			_, tmpExpected := modetest.PublishAllWith(t, output, test.events)
			expected := modetest.FlattenEvents(tmpExpected)

			// check we can find all event in topic
			timeout := 20 * time.Second
			stored := testReadFromKafkaTopic(t, test.topic, len(expected), timeout)

			// validate messages
			if len(expected) != len(stored) {
				assert.Equal(t, len(stored), len(expected))
				return
			}

			for i, d := range expected {
				var decoded map[string]interface{}
				err := json.Unmarshal(stored[i].Value, &decoded)
				if err != nil {
					t.Errorf("can not json decode event value: %v", stored[i].Value)
					return
				}
				event := d.Event

				assert.Equal(t, decoded["type"], event["type"])
				assert.Equal(t, decoded["message"], event["message"])
			}
		}()
	}
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func getTestKafkaHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("KAFKA_HOST", kafkaDefaultHost),
		getenv("KAFKA_PORT", kafkaDefaultPort),
	)
}

func makeConfig(t *testing.T, in map[string]interface{}) *common.Config {
	cfg, err := common.NewConfigFrom(in)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func newTestConsumer(t *testing.T) sarama.Consumer {
	hosts := []string{getTestKafkaHost()}
	consumer, err := sarama.NewConsumer(hosts, nil)
	if err != nil {
		t.Fatal(err)
	}
	return consumer
}

var testTopicOffsets = map[string]int64{}

func testReadFromKafkaTopic(
	t *testing.T, topic string, nMessages int,
	timeout time.Duration,
) []*sarama.ConsumerMessage {

	consumer := newTestConsumer(t)
	defer func() {
		consumer.Close()
	}()

	offset, found := testTopicOffsets[topic]
	if !found {
		offset = sarama.OffsetOldest
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, offset)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		partitionConsumer.Close()
	}()

	timer := time.After(timeout)
	var messages []*sarama.ConsumerMessage
	for i := 0; i < nMessages; i++ {
		select {
		case msg := <-partitionConsumer.Messages():
			messages = append(messages, msg)
			testTopicOffsets[topic] = msg.Offset + 1
		case <-timer:
			break
		}
	}

	return messages
}

func randMulti(batches, n int, event common.MapStr) []modetest.EventInfo {
	var out []modetest.EventInfo
	for i := 0; i < batches; i++ {
		var data []outputs.Data
		for j := 0; j < n; j++ {
			tmp := common.MapStr{}
			for k, v := range event {
				tmp[k] = v
			}
			tmp["message"] = randString(100)
			data = append(data, outputs.Data{Event: tmp})
		}

		out = append(out, modetest.EventInfo{Single: false, Data: data})
	}
	return out
}
