// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one

// or more contributor license agreements. Licensed under the Elastic License;

// you may not use this file except in compliance with the Elastic License.

//go:build windows

package resources

import (
	"testing"
)

func TestGUID(t *testing.T) {
	tests := []struct {
		name string
		bytes []byte
		want string
		knownFolder string
		expectError bool
	}{
		{
			name: "Videos",
			bytes: []byte{0x1d, 0x9b, 0x98, 0x18, 0xb5, 0x99, 0x5b, 0x45, 0x84, 0x1c, 0xab, 0x7c, 0x74, 0xe4, 0xdd, 0xfc},
			want: "18989B1D-99B5-455B-841C-AB7C74E4DDFC",
			knownFolder: "My Video",
			expectError: false,
		},
		{
			name: "Windows",
			bytes: []byte{0x04, 0xf4, 0x8b, 0xf3, 0x43, 0x1d, 0xf2, 0x42, 0x93, 0x05, 0x67, 0xde, 0x0b, 0x28, 0xfc, 0x23},
			want: "F38BF404-1D43-42F2-9305-67DE0B28FC23",
			knownFolder: "Windows",
			expectError: false,
		},
		{
			name: "Extra Bytes",
			bytes: []byte{0x1d, 0x9b, 0x98, 0x18, 0xb5, 0x99, 0x5b, 0x45, 0x84, 0x1c, 0xab, 0x7c, 0x74, 0xe4, 0xdd, 0xfc, 0x00, 0x00, 0x00, 0x00},
			want: "18989B1D-99B5-455B-841C-AB7C74E4DDFC",
			knownFolder: "My Video",
			expectError: true,
		},
	}
	for _, tt := range tests {
		guid, err := NewGUID(tt.bytes)
		if err != nil {
			if !tt.expectError {
				t.Errorf("NewGUID() error = %v", err)
			}
			continue
		}
		if got := guid.String(); got != tt.want {
			t.Errorf("GUID.String() = %v, want %v", got, tt.want)
		}
		knownFolder, ok := guid.LookupKnownFolder()
		if !ok {
			t.Errorf("GUID.LookupKnownFolder() = false, want true")
		}
		if knownFolder != tt.knownFolder {
			t.Errorf("GUID.LookupKnownFolder() = %v, want %v", knownFolder, tt.knownFolder)
		}
	}
}
