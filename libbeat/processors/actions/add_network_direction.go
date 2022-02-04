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

package actions

import (
	"fmt"
	"net"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("add_network_direction",
		checks.ConfigChecked(NewAddNetworkDirection,
			checks.RequireFields("source", "destination", "target", "internal_networks"),
			checks.AllowedFields("source", "destination", "target", "internal_networks")))
	jsprocessor.RegisterPlugin("AddNetworkDirection", NewAddNetworkDirection)
}

const (
	directionInternal = "internal"
	directionExternal = "external"
	directionOutbound = "outbound"
	directionInbound  = "inbound"
)

type networkDirectionProcessor struct {
	Source           string   `config:"source"`
	Destination      string   `config:"destination"`
	Target           string   `config:"target"`
	InternalNetworks []string `config:"internal_networks"`
}

// NewAddNetworkDirection constructs a new network direction processor.
func NewAddNetworkDirection(cfg *common.Config) (processors.Processor, error) {
	networkDirection := &networkDirectionProcessor{}
	if err := cfg.Unpack(networkDirection); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the add_network_direction configuration")
	}

	return networkDirection, nil
}

func (m *networkDirectionProcessor) Run(event *beat.Event) (*beat.Event, error) {
	sourceI, err := event.GetValue(m.Source)
	if err != nil {
		// doesn't have the required field value to analyze
		return event, nil
	}
	source, _ := sourceI.(string)
	if source == "" {
		// wrong type or not set
		return event, nil
	}
	destinationI, err := event.GetValue(m.Destination)
	if err != nil {
		// doesn't have the required field value to analyze
		return event, nil
	}
	destination, _ := destinationI.(string)
	if destination == "" {
		// wrong type or not set
		return event, nil
	}
	sourceIP := net.ParseIP(source)
	destinationIP := net.ParseIP(destination)
	if sourceIP == nil || destinationIP == nil {
		// bad ip address
		return event, nil
	}

	internalSource, err := conditions.NetworkContains(sourceIP, m.InternalNetworks...)
	if err != nil {
		return event, err
	}
	internalDestination, err := conditions.NetworkContains(destinationIP, m.InternalNetworks...)
	if err != nil {
		return event, err
	}

	event.Fields.DeepUpdate(common.MapStr{
		m.Target: networkDirection(internalSource, internalDestination),
	})
	return event, nil
}

func networkDirection(internalSource, internalDestination bool) string {
	if internalSource && internalDestination {
		return directionInternal
	}
	if internalSource {
		return directionOutbound
	}
	if internalDestination {
		return directionInbound
	}
	return directionExternal
}

func (m *networkDirectionProcessor) String() string {
	return fmt.Sprintf("networkDirection=%+v|%+v->%+v", m.Source, m.Destination, m.Target)
}
