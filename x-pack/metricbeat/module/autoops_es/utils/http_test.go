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

type FakeObject struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func TestDeserializeDataFails(t *testing.T) {
	var invalidJson = []string{`{"id":true}`, `{"malformed"}`}

	for _, json := range invalidJson {
		_, err := DeserializeData[FakeObject]([]byte(json))

		require.ErrorContains(t, err, "failed to deserialize data")
	}
}

func TestDeserializeDataSucceeds(t *testing.T) {
	var validJson = []string{`{}`, `{"id": "123"}`, `{"id":"456","name":"the name","other":"field"}`}

	for _, json := range validJson {
		obj, err := DeserializeData[FakeObject]([]byte(json))

		require.NoError(t, err)
		require.NotNil(t, obj)
	}

	obj, err := DeserializeData[FakeObject]([]byte(validJson[2]))

	require.NoError(t, err)
	require.Equal(t, "456", obj.Id)
	require.Equal(t, "the name", obj.Name)
}
