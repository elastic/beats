// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/libbeat/management/api"
)

func TestErrorSerialization(t *testing.T) {
	id, _ := uuid.NewV4()
	t.Run("serialize ok", func(t *testing.T) {
		e := Error{
			Type: ConfigError,
			Err:  errors.New("hello world"),
			UUID: id,
		}

		b, err := json.Marshal(&e)
		if assert.NoError(t, err) {
			return
		}

		resp := &struct {
			UUID    string `json:"uuid"`
			Message string `json:"message"`
			Type    string `json:"type"`
		}{}

		err = json.Unmarshal(b, resp)
		if assert.NoError(t, err) {
			return
		}

		assert.Equal(t, e.UUID.String(), resp.UUID)
		assert.Equal(t, e.Err.Error(), resp.Message)
		assert.Equal(t, e.Type, api.EventType(resp.Type))
	})

	t.Run("ensure that json general fields are present", ensureJSONhasGeneralfield(t, &Error{
		Type: ConfigError,
		Err:  errors.New("hello world"),
		UUID: id,
	}))
}

func TestErrors(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errors := Errors{newConfigError(errors.New("error1"))}
		assert.Equal(t, "1 error: error1", errors.Error())
	})

	t.Run("multiple errors", func(t *testing.T) {
		errors := Errors{
			newConfigError(errors.New("error1")),
			newConfigError(errors.New("error2")),
		}
		assert.Equal(t, "2 errors: error1; error2", errors.Error())
	})
}
