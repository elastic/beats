// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReduceTimestampPrecision(t *testing.T) {
	oneSecond := time.Second.Nanoseconds()
	result1 := ReduceTimestampPrecision(uint64(oneSecond))
	require.Equal(t, time.Duration(oneSecond), result1)

	oneSecondWithDelay := oneSecond + 10
	result2 := ReduceTimestampPrecision(uint64(oneSecondWithDelay))
	require.Equal(t, time.Duration(oneSecond), result2)
}
