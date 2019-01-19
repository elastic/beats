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

package pulsar

import (
    "context"
    "encoding/json"
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"

    "github.com/apache/pulsar/pulsar-client-go/pulsar"
    "github.com/elastic/beats/libbeat/beat"
    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/outputs"
    _ "github.com/elastic/beats/libbeat/outputs/codec/format"
    _ "github.com/elastic/beats/libbeat/outputs/codec/json"
    "github.com/elastic/beats/libbeat/outputs/outest"
)

type eventInfo struct {
    events []beat.Event
}

func makeConfig(t *testing.T, in map[string]interface{}) *common.Config {
    cfg, err := common.NewConfigFrom(in)
    if err != nil {
        t.Fatal(err)
    }
    return cfg
}

func single(fields common.MapStr) []eventInfo {
    return []eventInfo{
        {
            events: []beat.Event{
                {Timestamp: time.Now(), Fields: fields},
            },
        },
    }
}

func flatten(infos []eventInfo) []beat.Event {
    var out []beat.Event
    for _, info := range infos {
        out = append(out, info.events...)
    }
    return out
}

func TestPulsarPublish(t *testing.T) {
    pulsarConfig := map[string]interface{}{
        "url":           "pulsar://pulsar-authentication:6650",
        "io_threads":    5,
        "topic":         "my_topic",
        "bulk_max_size": 2048,
        "max_retries":   3,
    }
    testPulsarPublishMessage(t, pulsarConfig)
}

func testPulsarPublishMessage(t *testing.T, cfg map[string]interface{}) {

    tests := []struct {
        title  string
        config map[string]interface{}
        topic  string
        events []eventInfo
    }{
        {
            "test single events",
            map[string]interface{}{
                "url":                       "pulsar+ssl://pulsar-authentication:6651",
                "topic":                     "my-topic1",
                "name":                      "test",
                "use_tls":                   true,
                "tls_trust_certs_file_path": "/go/src/github.com/AmateurEvents/filebeat-ouput-pulsar/certs/ca.cert.pem",
                "certificate_path":          "/go/src/github.com/AmateurEvents/filebeat-ouput-pulsar/role/admin.cert.pem",
                "private_key_path":          "/go/src/github.com/AmateurEvents/filebeat-ouput-pulsar/role/admin.key-pk8.pem",
            },
            "my-topic1",
            single(common.MapStr{
                "type":    "log",
                "message": "test123",
            }),
        },
    }
    for i, test := range tests {
        config := makeConfig(t, cfg)
        if test.config != nil {
            config.Merge(makeConfig(t, test.config))
        }
        name := fmt.Sprintf("run test(%v): %v", i, test.title)
        t.Run(name, func(t *testing.T) {
            grp, err := makePulsar(beat.Info{Beat: "libbeat"}, outputs.NewNilObserver(), config)
            if err != nil {
                t.Fatal(err)
            }

            output := grp.Clients[0].(*client)
            if err := output.Connect(); err != nil {
                t.Fatal(err)
            }
            defer output.Close()
            // publish test events
            for i := range test.events {
                batch := outest.NewBatch(test.events[i].events...)

                output.Publish(batch)
            }

            expected := flatten(test.events)

            stored := testReadFromPulsarTopic(t, output.clientOptions, test.topic, len(expected))
            for i, d := range expected {
                validateJSON(t, stored[i].Payload(), d)
            }
        })
    }
}

func testReadFromPulsarTopic(
    t *testing.T, clientOptions pulsar.ClientOptions,
    topic string, nMessages int) []pulsar.Message {
    // Instantiate a Pulsar client
    client, err := pulsar.NewClient(clientOptions)

    if err != nil {
        t.Fatal(err)
    }

    // Use the client object to instantiate a consumer
    consumer, err := client.Subscribe(pulsar.ConsumerOptions{
        Topic:            topic,
        SubscriptionName: "sub-1",
        Type:             pulsar.Shared,
    })

    if err != nil {
        t.Fatal(err)
    }

    defer consumer.Close()

    ctx := context.Background()
    var messages []pulsar.Message
    for i := 0; i < nMessages; i++ {
        msg, err := consumer.Receive(ctx)
        if err != nil {
            t.Fatal(err)
        }

        // Do something with the message

        consumer.Ack(msg)
        messages = append(messages, msg)
    }
    return messages
}

func validateJSON(t *testing.T, value []byte, event beat.Event) {
    var decoded map[string]interface{}
    err := json.Unmarshal(value, &decoded)
    if err != nil {
        t.Errorf("can not json decode event value: %v", value)
        return
    }
    assert.Equal(t, decoded["type"], event.Fields["type"])
    assert.Equal(t, decoded["message"], event.Fields["message"])
}
