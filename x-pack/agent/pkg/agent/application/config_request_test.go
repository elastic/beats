// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigRequest(t *testing.T) {
	t.Run("limit case for ShortID()", func(t *testing.T) {
		c := configRequest{id: "bye"}
		require.Equal(t, "bye", c.ShortID())

		// TODO(PH): add validation when we create the config request.
		c = configRequest{id: ""}
		require.Equal(t, "", c.ShortID())
	})

	t.Run("ShortID()", func(t *testing.T) {
		c := configRequest{id: "HELLOWORLDBYEBYE"}
		require.Equal(t, "HELLOWOR", c.ShortID())
	})
}
