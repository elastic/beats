// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"
	"time"
)

func Test_convertFileTimeToGoTime(t *testing.T) {
	tests := []struct {
		name     string
		fileTime uint64
		want     time.Time
	}{
		{
			name:     "TestZeroValue",
			fileTime: 0,
			want:     time.Time{},
		},
		{
			name:     "TestUnixEpoch",
			fileTime: 116444736000000000, // January 1, 1970 (Unix epoch)
			want:     time.Unix(0, 0),
		},
		{
			name:     "TestActualDate",
			fileTime: 133515900000000000, // February 05, 2024, 7:00:00 AM
			want:     time.Date(2024, 0o2, 0o5, 7, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFileTimeToGoTime(tt.fileTime)
			if !got.Equal(tt.want) {
				t.Errorf("convertFileTimeToGoTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLazyCacheMapValue(t *testing.T) {
	// Create a mock cached map info with lazy cache
	cachedMapInfo := &cachedEventMapInfo{
		Name:                   "TestMap",
		formattedCachedEntries: make(map[string]formattedMapCacheEntry),
	}

	// Create a mock property info
	propInfo := &cachedPropertyInfo{
		InType:  TdhIntypeUint32,
		MapName: "TestMap",
	}

	// Test data - a 4-byte integer
	testData := []byte{0x01, 0x02, 0x03, 0x04}
	expectedResult := "Test Result"
	expectedConsumed := 4

	// Cache a value
	cachedMapInfo.cacheFormattedMapEntry(propInfo, testData, expectedResult, expectedConsumed)

	// Verify it was cached
	if len(cachedMapInfo.formattedCachedEntries) != 1 {
		t.Errorf("Expected 1 cached entry, got %d", len(cachedMapInfo.formattedCachedEntries))
	}

	// Try to get the cached value
	result, consumed, ok := cachedMapInfo.getFormattedMapEntry(propInfo, testData, expectedConsumed)
	if !ok {
		t.Error("Expected cached value to be found")
	}

	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	if consumed != expectedConsumed {
		t.Errorf("Expected consumed %d, got %d", expectedConsumed, consumed)
	}
}

func TestLazyCacheMapValueEdgeCases(t *testing.T) {
	// Create a mock cached map info with lazy cache
	cachedMapInfo := &cachedEventMapInfo{
		Name:                   "TestMap",
		formattedCachedEntries: make(map[string]formattedMapCacheEntry),
	}

	// Create a mock property info
	propInfo := &cachedPropertyInfo{
		InType:  TdhIntypeUint32,
		MapName: "TestMap",
	}

	// Test case 1: Cache miss - no entry should be found
	testData := []byte{0x01, 0x02, 0x03, 0x04}
	result, consumed, ok := cachedMapInfo.getFormattedMapEntry(propInfo, testData, 4)
	if ok {
		t.Error("Expected cache miss for new data, but found cached value")
	}

	// Test case 2: Cache a value and verify it's retrievable
	expectedResult := "Test Result"
	expectedConsumed := 4
	cachedMapInfo.cacheFormattedMapEntry(propInfo, testData, expectedResult, expectedConsumed)

	result, consumed, ok = cachedMapInfo.getFormattedMapEntry(propInfo, testData, 4)
	if !ok {
		t.Error("Expected cached value to be found after caching")
	}
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}
	if consumed != expectedConsumed {
		t.Errorf("Expected consumed %d, got %d", expectedConsumed, consumed)
	}

	// Test case 3: Different data should result in cache miss
	differentData := []byte{0x05, 0x06, 0x07, 0x08}
	result, consumed, ok = cachedMapInfo.getFormattedMapEntry(propInfo, differentData, 4)
	if ok {
		t.Error("Expected cache miss for different data, but found cached value")
	}

	// Test case 4: Different map name should result in cache miss
	differentPropInfo := &cachedPropertyInfo{
		InType:  TdhIntypeUint32,
		MapName: "DifferentMap",
	}
	result, consumed, ok = cachedMapInfo.getFormattedMapEntry(differentPropInfo, testData, 4)
	if ok {
		t.Error("Expected cache miss for different map name, but found cached value")
	}

	// Test case 5: Multiple entries in cache
	cachedMapInfo.cacheFormattedMapEntry(propInfo, differentData, "Different Result", 4)

	// Verify both entries exist
	if len(cachedMapInfo.formattedCachedEntries) != 2 {
		t.Errorf("Expected 2 cached entries, got %d", len(cachedMapInfo.formattedCachedEntries))
	}

	// Verify both can be retrieved
	result1, _, ok1 := cachedMapInfo.getFormattedMapEntry(propInfo, testData, 4)
	result2, _, ok2 := cachedMapInfo.getFormattedMapEntry(propInfo, differentData, 4)

	if !ok1 || !ok2 {
		t.Error("Expected both cached values to be found")
	}
	if result1 != "Test Result" || result2 != "Different Result" {
		t.Errorf("Expected cached results to match, got %q and %q", result1, result2)
	}
}
