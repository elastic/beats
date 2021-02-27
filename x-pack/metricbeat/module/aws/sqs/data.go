// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sqs

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var (
	schemaRequestFields = s.Schema{
		"oldest_message_age": s.Object{
			"sec": c.Float("ApproximateAgeOfOldestMessage"),
		},
		"messages": s.Object{
			"delayed":     c.Float("ApproximateNumberOfMessagesDelayed"),
			"not_visible": c.Float("ApproximateNumberOfMessagesNotVisible"),
			"visible":     c.Float("ApproximateNumberOfMessagesVisible"),
			"deleted":     c.Float("NumberOfMessagesDeleted"),
			"received":    c.Float("NumberOfMessagesReceived"),
			"sent":        c.Float("NumberOfMessagesSent"),
		},
		"empty_receives": c.Float("NumberOfEmptyReceives"),
		"sent_message_size": s.Object{
			"bytes": c.Float("SentMessageSize"),
		},
		"queue.name": c.Str("QueueName"),
	}
)
