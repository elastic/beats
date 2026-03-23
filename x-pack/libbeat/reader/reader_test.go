// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reader

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/klauspost/compress/snappy"
)

const testJSON = "{\"message\":\"hello, world!\",\"count\":42}\n"

func TestIsStreamGzipped(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "gzipped",
			data: gzCompress(t, []byte(testJSON)),
			want: true,
		},
		{
			name: "plain text",
			data: []byte(testJSON),
			want: false,
		},
		{
			name: "snappy",
			data: snappyCompress(t, []byte(testJSON)),
			want: false,
		},
		{
			name: "empty",
			data: []byte{},
			want: false,
		},
		{
			name: "too short",
			data: []byte{0x1f},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(tt.data))
			got, err := IsStreamGzipped(r)
			if err != nil {
				t.Fatalf("IsStreamGzipped returned unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("IsStreamGzipped = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStreamSnappy(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "snappy",
			data: snappyCompress(t, []byte(testJSON)),
			want: true,
		},
		{
			name: "plain text",
			data: []byte(testJSON),
			want: false,
		},
		{
			name: "gzipped",
			data: gzCompress(t, []byte(testJSON)),
			want: false,
		},
		{
			name: "empty",
			data: []byte{},
			want: false,
		},
		{
			name: "too short",
			data: []byte{0xff, 0x06},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(tt.data))
			got, err := IsStreamSnappy(r)
			if err != nil {
				t.Fatalf("IsStreamSnappy returned unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("IsStreamSnappy = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddDecoderIfNeeded(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "plain text",
			data: []byte(testJSON),
			want: testJSON,
		},
		{
			name: "gzipped",
			data: gzCompress(t, []byte(testJSON)),
			want: testJSON,
		},
		{
			name: "snappy",
			data: snappyCompress(t, []byte(testJSON)),
			want: testJSON,
		},
		{
			name: "empty",
			data: []byte{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := AddDecoderIfNeeded(bytes.NewReader(tt.data))
			if err != nil {
				t.Fatalf("AddDecoderIfNeeded returned unexpected error: %v", err)
			}

			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("io.ReadAll returned unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("decoded stream = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func gzCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("gzip write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}
	return buf.Bytes()
}

func snappyCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := snappy.NewBufferedWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("snappy write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("snappy close failed: %v", err)
	}
	return buf.Bytes()
}
