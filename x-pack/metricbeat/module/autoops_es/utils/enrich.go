// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

type EnrichedType[T any] struct {
	// This is never called if either diff is 0; instead it will simply write 0
	CalculateValue func(timeDiff float64, valueDiff int64) float64
	ConvertTime    func(int64) float64
	GetTime        func(*T, int64) int64
	GetValue       func(*T) int64
	IsUsable       func(*T) bool
	WriteValue     func(*T, float64)
}

type EnrichedCache[T any] struct {
	Enrichers []EnrichedType[T]

	// Latest timestamp _now_
	NewTimestamp      int64
	PreviousTimestamp int64
	PreviousCache     map[string]T
}

func CalculateLatency(timeDiff float64, valueDiff int64) float64 {
	return timeDiff / float64(valueDiff)
}

func CalculateRate(timeDiff float64, valueDiff int64) float64 {
	return float64(valueDiff) / timeDiff
}

func MillisToSeconds(millis int64) float64 {
	return float64(millis) / 1000
}

func UseTimestamp[T any](_ T, timestamp int64) int64 {
	return timestamp
}

func UseTimeInMillis(millis int64) float64 {
	return float64(millis)
}

// Enrich the `obj` by invoking each `EnrichedType` against the `obj` and `prevObj` using details from the `cache` to add a rate or latency value to the `obj`.
func EnrichObject[T any](obj *T, prevObj *T, cache EnrichedCache[T]) {
	if obj == nil || prevObj == nil {
		return
	}

	for _, enricher := range cache.Enrichers {
		if !enricher.IsUsable(obj) || !enricher.IsUsable(prevObj) {
			continue
		}

		newTime := enricher.GetTime(obj, cache.NewTimestamp)
		newValue := enricher.GetValue(obj)

		prevTime := enricher.GetTime(prevObj, cache.PreviousTimestamp)
		prevValue := enricher.GetValue(prevObj)

		if newTime >= prevTime && newValue >= prevValue {
			timeDiff := enricher.ConvertTime(newTime - prevTime)
			valueDiff := newValue - prevValue

			calculated := float64(0)

			// either being zero means there's no measurable value exists
			if timeDiff > 0 && valueDiff > 0 {
				calculated = enricher.CalculateValue(timeDiff, valueDiff)
			}

			enricher.WriteValue(obj, calculated)
		}
	}
}
