// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializationOfState(t *testing.T) {
	t.Run("serialize ok", func(t *testing.T) {
		e := &Starting

		b, err := json.Marshal(&e)
		if assert.NoError(t, err) {
			return
		}

		resp := &struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		}{}

		err = json.Unmarshal(b, resp)
		if assert.NoError(t, err) {
			return
		}

		assert.Equal(t, e.String(), resp.Type)
		assert.NotEmpty(t, resp.Message)
	})
	t.Run("ensure that json general fields are present", ensureJSONhasGeneralfield(t, &Starting))
}

// Ensure that all events have a Message key that can by used by the GUI.
func ensureJSONhasGeneralfield(t *testing.T, obj json.Marshaler) func(*testing.T) {
	return func(t *testing.T) {
		serialized, err := json.Marshal(obj)
		if !assert.NoError(t, err) {
			return
		}

		message := struct {
			Message string `json:"message"`
		}{}

		err = json.Unmarshal(serialized, &message)

		if !assert.NoError(t, err) {
			return
		}
		assert.NotEmpty(t, message)
	}
}
