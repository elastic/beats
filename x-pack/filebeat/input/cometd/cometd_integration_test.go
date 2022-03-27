// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"testing"

	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/stretchr/testify/require"
)

func TestInput(t *testing.T) {
	err := input.Register(inputName, NewInput)
	require.Error(t, err)

	var cometd cometdInput
	cometd.Run()

	makeEvent("test", "test")
}
