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
