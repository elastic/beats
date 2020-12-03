// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitEventsBy(t *testing.T) {
	event := map[string]interface{}{
		"this": "is kept",
		"alerts": []interface{}{
			map[string]interface{}{
				"this_is": "also kept",
				"entities": []interface{}{
					map[string]interface{}{
						"something": "something",
					},
					map[string]interface{}{
						"else": "else",
					},
				},
			},
			map[string]interface{}{
				"this_is": "also kept 2",
				"entities": []interface{}{
					map[string]interface{}{
						"something": "something 2",
					},
					map[string]interface{}{
						"else": "else 2",
					},
				},
			},
		},
	}

	expectedEvents := []map[string]interface{}{
		{
			"this": "is kept",
			"alerts": map[string]interface{}{
				"this_is": "also kept",
				"entities": map[string]interface{}{
					"something": "something",
				},
			},
		},
		{
			"this": "is kept",
			"alerts": map[string]interface{}{
				"this_is": "also kept",
				"entities": map[string]interface{}{
					"else": "else",
				},
			},
		},
		{
			"this": "is kept",
			"alerts": map[string]interface{}{
				"this_is": "also kept 2",
				"entities": map[string]interface{}{
					"something": "something 2",
				},
			},
		},
		{
			"this": "is kept",
			"alerts": map[string]interface{}{
				"this_is": "also kept 2",
				"entities": map[string]interface{}{
					"else": "else 2",
				},
			},
		},
	}

	const key = "alerts..entities"

	got := splitEvent(key, event)

	assert.Equal(t, expectedEvents, got)
}
