// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		errors := Errors{NewConfigError(errors.New("error1"))}
		assert.Equal(t, "1 error: error1", errors.Error())
	})

	t.Run("multiple errors", func(t *testing.T) {
		errors := Errors{
			NewConfigError(errors.New("error1")),
			NewConfigError(errors.New("error2")),
		}
		assert.Equal(t, "2 errors: error1; error2", errors.Error())
	})
}
