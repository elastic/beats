// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sqs

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaRequestFields = s.Schema{
		"oldest_message_age": s.Object{
			"sec": c.Float("ApproximateAgeOfOldestMessage"),
		},
		"messages": s.Object{
			"delayed": s.Object{
				"count": c.Int("ApproximateNumberOfMessagesDelayed"),
			},
			"not_visible": s.Object{
				"count": c.Int("ApproximateNumberOfMessagesNotVisible"),
			},
			"visible": s.Object{
				"count": c.Int("ApproximateNumberOfMessagesVisible"),
			},
			"deleted": s.Object{
				"count": c.Int("NumberOfMessagesDeleted"),
			},
			"received": s.Object{
				"count": c.Int("NumberOfMessagesReceived"),
			},
			"sent": s.Object{
				"count": c.Int("NumberOfMessagesSent"),
			},
		},
		"empty_receives": s.Object{
			"count": c.Int("NumberOfEmptyReceives"),
		},
		"sent_message_size": s.Object{
			"bytes": c.Float("SentMessageSize"),
		},
	}
)
