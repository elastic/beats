// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package topic

// Topic is the identifier for a communication channel.
type Topic string

// AllSubscribers is a wildcard topic used to target all the Subscribes in the event bus.
const AllSubscribers Topic = "*"

// StateChanges is a topic for describing precise steps in order to achieve desired configuration.
const StateChanges Topic = "StateChanges"

// Configurations is a topic for communicating new desired configurations.
const Configurations Topic = "Configurations"
