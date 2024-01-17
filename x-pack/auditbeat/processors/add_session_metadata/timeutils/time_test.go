// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReduceTimestampPrecision(t *testing.T) {
	oneSecond := uint64(time.Second.Nanoseconds())
	result1 := ReduceTimestampPrecision(oneSecond)
	require.Equal(t, oneSecond, result1)

	oneSecondWithDelay := oneSecond + 10
	result2 := ReduceTimestampPrecision(oneSecondWithDelay)
	require.Equal(t, oneSecond, result2)
}
