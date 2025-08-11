// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUUIDV4(t *testing.T) {
	uuid := NewUUIDV4()

	require.NotEmpty(t, uuid)
	require.NotEqual(t, uuid, NewUUIDV4())
}
