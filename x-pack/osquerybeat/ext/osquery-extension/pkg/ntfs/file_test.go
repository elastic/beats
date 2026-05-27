// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package ntfs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"www.velocidex.com/golang/go-ntfs/parser"
)

// makeFileName builds a *parser.FILE_NAME whose NameType byte equals namespace.
// Off_FILE_NAME_NameType = 65 per NewNTFSProfile() (NTFS $FILE_NAME attribute layout:
// 8 bytes parent ref, 32 bytes timestamps, 8 bytes sizes, 4 flags, 4 reparse,
// 1 name-length, then 1 namespace byte).
func makeFileName(t *testing.T, namespace byte) *parser.FILE_NAME {
	t.Helper()
	buf := make([]byte, 128)
	buf[65] = namespace
	return parser.NewNTFSProfile().FILE_NAME(bytes.NewReader(buf), 0)
}

func TestPreferredFileName(t *testing.T) {
	const (
		nsPOSIX       = byte(0)
		nsWin32       = byte(1)
		nsDOS         = byte(2)
		nsWin32AndDOS = byte(3)
		wantNil       = byte(255) // sentinel: expect nil return
	)

	tests := []struct {
		name       string
		namespaces []byte
		wantNS     byte
	}{
		{
			name:       "nil slice returns nil",
			namespaces: nil,
			wantNS:     wantNil,
		},
		{
			name:       "empty slice returns nil",
			namespaces: []byte{},
			wantNS:     wantNil,
		},
		{
			name:       "Win32 only",
			namespaces: []byte{nsWin32},
			wantNS:     nsWin32,
		},
		{
			name:       "Win32+DOS only",
			namespaces: []byte{nsWin32AndDOS},
			wantNS:     nsWin32AndDOS,
		},
		{
			name:       "Win32 preferred over Win32+DOS",
			namespaces: []byte{nsWin32AndDOS, nsWin32},
			wantNS:     nsWin32,
		},
		{
			name:       "Win32+DOS preferred over POSIX",
			namespaces: []byte{nsPOSIX, nsWin32AndDOS},
			wantNS:     nsWin32AndDOS,
		},
		{
			name:       "POSIX preferred over DOS",
			namespaces: []byte{nsDOS, nsPOSIX},
			wantNS:     nsPOSIX,
		},
		{
			name:       "DOS is last resort",
			namespaces: []byte{nsDOS},
			wantNS:     nsDOS,
		},
		{
			name:       "Win32 beats all other namespaces",
			namespaces: []byte{nsDOS, nsPOSIX, nsWin32AndDOS, nsWin32},
			wantNS:     nsWin32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fns []*parser.FILE_NAME
			for _, ns := range tt.namespaces {
				fns = append(fns, makeFileName(t, ns))
			}

			got := preferredFileName(fns)

			if tt.wantNS == wantNil {
				assert.Nil(t, got, "expected nil for empty/nil input")
				return
			}
			require.NotNil(t, got, "expected non-nil FILE_NAME")
			assert.Equal(t, uint64(tt.wantNS), got.NameType().Value, "wrong namespace selected")
		})
	}
}
