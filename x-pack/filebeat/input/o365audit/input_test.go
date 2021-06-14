// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestPreserveOriginalEvent(t *testing.T) {
	env := apiEnvironment{
		Config: APIConfig{PreserveOriginalEvent: false},
	}

	doc := common.MapStr{
		"field1": "val1",
	}

	event := env.toBeatEvent(doc)

	v, err := event.GetValue("event.original")
	require.EqualError(t, err, "key not found")
	assert.Nil(t, v)

	env.Config.PreserveOriginalEvent = true

	event = env.toBeatEvent(doc)

	v, err = event.GetValue("event.original")
	require.NoError(t, err)
	assert.JSONEq(t, `{"field1":"val1"}`, v.(string))
}
