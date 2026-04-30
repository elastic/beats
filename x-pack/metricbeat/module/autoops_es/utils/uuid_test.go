// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package utils

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewUUID(t *testing.T) {
	uuid := NewUUID()

	require.NotEmpty(t, uuid)
	require.Len(t, uuid, 32) // UUID v7 without dashes should be 32 characters long
	require.NotEqual(t, uuid, NewUUID())
}

func TestNewUUIDIsUUIDv7(t *testing.T) {
	uuid := NewUUID()

	// UUID v7 has the format xxxxxxxx-xxxx-7xxx-yxxx-xxxxxxxxxxxx where y is one of [8, 9, A, B]
	// Since our implementation removes dashes, the version character is at index 12 and the variant character is at index 16.
	require.EqualValues(t, '7', uuid[12])    // Version should be '7'
	require.Contains(t, "89ab", uuid[16:17]) // Variant should be one of '8', '9', 'A', or 'B'

	// UUIDv7 encodes a Unix millisecond timestamp in the first 48 bits (12 hex chars).
	// We can take the first 12 characters because our implementation removes dashes.
	tsMillis, err := strconv.ParseInt(uuid[:12], 16, 64)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), time.UnixMilli(tsMillis), 10*time.Minute)
}
