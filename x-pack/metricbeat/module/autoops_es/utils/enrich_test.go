// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type cachedObject struct {
	IndexLatencyInMillis *float64
	IndexRatePerSecond   *float64
	IndexingTotal        *int64
	IndexingTotalTime    *int64
}

var (
	cache = EnrichedCache[cachedObject]{
		Enrichers: []EnrichedType[cachedObject]{
			// RATE EXAMPLE:
			{
				CalculateValue: CalculateRate,
				ConvertTime:    MillisToSeconds,
				GetTime:        UseTimestamp[*cachedObject],
				GetValue:       func(obj *cachedObject) int64 { return *obj.IndexingTotal },
				IsUsable:       func(obj *cachedObject) bool { return obj.IndexingTotal != nil },
				WriteValue:     func(obj *cachedObject, value float64) { obj.IndexRatePerSecond = &value },
			},
			// LATENCY EXAMPLE:
			{
				CalculateValue: CalculateLatency,
				ConvertTime:    UseTimeInMillis,
				GetTime:        func(obj *cachedObject, _ int64) int64 { return *obj.IndexingTotalTime },
				GetValue:       func(obj *cachedObject) int64 { return *obj.IndexingTotal },
				IsUsable: func(obj *cachedObject) bool {
					return obj.IndexingTotal != nil && obj.IndexingTotalTime != nil
				},
				WriteValue: func(obj *cachedObject, value float64) { obj.IndexLatencyInMillis = &value },
			},
		},
	}
)

func clearCache() {
	cache.PreviousCache = nil
	cache.PreviousTimestamp = 0

	newCache()
}

func initCache(previousSeconds int64) {
	newCache()

	cache.PreviousCache = map[string]cachedObject{}
	cache.PreviousTimestamp = cache.NewTimestamp - (previousSeconds * 1_000)
}

func newCache() {
	cache.NewTimestamp = time.Now().UnixMilli()
}

func object(indexingTotal int64, indexingTotalTime int64) cachedObject {
	return cachedObject{
		IndexingTotal:     &indexingTotal,
		IndexingTotalTime: &indexingTotalTime,
	}
}

func TestUncached(t *testing.T) {
	clearCache()

	data := object(10, 1)

	EnrichObject(&data, nil, cache)

	require.Nil(t, data.IndexRatePerSecond)
	require.Nil(t, data.IndexLatencyInMillis)
}

func TestCacheMiss(t *testing.T) {
	initCache(10)

	data := object(10, 1)

	EnrichObject(&data, nil, cache)

	require.Nil(t, data.IndexRatePerSecond)
	require.Nil(t, data.IndexLatencyInMillis)
}

func TestCacheHit(t *testing.T) {
	initCache(10)

	prevData := object(10, 1)
	data := object(20, 2)

	EnrichObject(&data, &prevData, cache)

	// 1 per second
	require.EqualValues(t, 1, *data.IndexRatePerSecond)
	require.EqualValues(t, 0.1, *data.IndexLatencyInMillis)

	nextData := object(30, 3)

	EnrichObject(&nextData, &data, cache)

	// 1 per second
	require.EqualValues(t, 1, *data.IndexRatePerSecond)
	require.EqualValues(t, 0.1, *data.IndexLatencyInMillis)
}
