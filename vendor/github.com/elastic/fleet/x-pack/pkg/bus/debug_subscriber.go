// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bus

import (
	"fmt"

	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
)

// AddMultipleSubscribersToTopics adds the same subscriber function to multiples topics, will return
// an error if we cannot subscribe to the specific topic.
func AddMultipleSubscribersToTopics(fn SubscribeFunc, bus Bus, topics ...topic.Topic) error {
	for _, topic := range topics {
		if err := bus.Subscribe(topic, fn); err != nil {
			return err
		}
	}
	return nil
}

// MustAddMultipleSubscribersToTopics adds the same subscriber function to multiples topics, will panic
// on any errors.
func MustAddMultipleSubscribersToTopics(fn SubscribeFunc, bus Bus, topics ...topic.Topic) {
	if err := AddMultipleSubscribersToTopics(fn, bus, topics...); err != nil {
		panic(fmt.Sprintf("could not add multiples subscribers, error: %+v", err))
	}
}
