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
	"errors"
	"fmt"
)

type azureInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	Namespace                []string `config:"namespace" validate:"required"`
	EventHubs                []string `config:"eventhub" validate:"required"`
	ConsumerGroup            string   `config:"consumer_group" validate:"required"`
	ConnectionStringValue    string   `config:"connection_string" validate:"required"`
	ExpandEventListFromField string   `config:"expand_event_list_from_field"`
}

// The default config for the kafka input. When in doubt, default values
// were chosen to match sarama's defaults.
func defaultConfig() azureInputConfig {
	return azureInputConfig{
		ExpandEventListFromField: "records",
	}
}

// Validate validates the config.
func (c *azureInputConfig) Validate() error {
	if len(c.Namespace) == 0 {
		return errors.New("no event hub namespace has been configured")
	}
	if len(c.EventHubs) == 0 {
		return errors.New("no event hubs have been configured")
	}
	if c.ConnectionStringValue == "" {
		return fmt.Errorf("no configuration string has been configured")
	}
	return nil
}
