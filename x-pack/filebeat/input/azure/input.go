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

package azure

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/filebeat/input/kafka"
	"github.com/elastic/beats/libbeat/beat"
	"strings"
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"
)

func init() {
	err := input.Register("azure", NewInput)
	if err != nil {
		panic(err)
	}
}

// NewInput creates a new kafka input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {

	// Wrap log input with custom docker settings
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "reading container input config")
	}

	err := cfg.Merge(common.MapStr{
		"hosts":                        config.Namespace,
		"topics":                       config.EventHubs,
		"group_id":                     config.ConsumerGroup,
		"username":                     "$ConnectionString",
		"password":                     config.ConnectionStringValue,
		"expand_event_list_from_field": config.ExpandEventListFromField,
		"ssl.enabled":                  true,
	})

	if err != nil {
		return nil, errors.Wrap(err, "update input config")
	}
	var eventFunc kafka.MapEvents = mapEvents
	dynFields := common.NewMapStrPointer(common.MapStr{
		"map_events": eventFunc,
	})
	inputContext.DynamicFields = &dynFields
	return kafka.NewInput(cfg, connector, inputContext)
}

func mapEvents(messages []string, kafkaFields common.MapStr) []beat.Event {
	var events []beat.Event
	for _, msg := range messages {
		//remove empty fields
		msg:= strings.ReplaceAll(msg, "\"\":\"\",", "")
		message, timestamp, err := parseMessage(msg)
		if err != nil {
			continue
		}
		event := beat.Event{
			Timestamp: timestamp,
			Fields: common.MapStr{},
		}
		for key, value := range message {
			if key != "" {
				event.Fields.Put(fmt.Sprintf("azure.%s", key), value)
			}
		}
		for key, value := range kafkaFields {
			if key != "" {
				event.Fields.Put(fmt.Sprintf("azure.input.%s", key), value)
			}
		}

		events = append(events, event)
	}
	return events
}

type Record struct {
	Time time.Time `json:"time"`
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func parseMessage(message string) (common.MapStr, time.Time, error) {
	var obj common.MapStr
	err := json.Unmarshal([]byte(message), &obj)
	if err != nil {
		return obj, time.Now(), err
	}
	var timestamp Record
	err = json.Unmarshal([]byte(message), &timestamp)
	if err != nil {
		return obj, time.Now(), err
	}
	return obj, timestamp.Time, nil
}
