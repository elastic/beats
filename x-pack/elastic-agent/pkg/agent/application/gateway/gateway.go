// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gateway

import "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"

// FleetGateway is a gateway between the Agent and the Fleet API, it's take cares of all the
// bidirectional communication requirements. The gateway aggregates events and will periodically
// call the API to send the events and will receive actions to be executed locally.
// The only supported action for now is a "ActionPolicyChange".
type FleetGateway interface {
	// Start starts the gateway.
	Start() error

	// Set the client for the gateway.
	SetClient(client.Sender)
}
