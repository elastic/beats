// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/client"
)

func TestParseEventMetadata(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected map[string]any
	}{
		{
			name: "complete metadata",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatNameCtxKey:    {"filebeat"},
						BeatVersionCtxKey: {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "filebeat",
				"version": "8.0.0",
			},
		},
		{
			name: "missing beat name",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatVersionCtxKey: {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "",
				"version": "8.0.0",
			},
		},
		{
			name: "missing beat version",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatNameCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "filebeat",
				"version": "",
			},
		},
		{
			name: "no metadata",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "",
				"version": "",
			},
		},
		{
			name: "no client info in context",
			setupCtx: func() context.Context {
				return t.Context()
			},
			expected: map[string]any{
				"beat":    "",
				"version": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			metadata := GetBeatEventMeta(ctx)

			assert.Equal(t, tt.expected, metadata)
		})
	}
}

func TestGetBeatVersion(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "version exists",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatNameCtxKey:    {"filebeat"},
						BeatVersionCtxKey: {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "8.0.0",
		},
		{
			name: "version missing",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatNameCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "",
		},
		{
			name: "no client info",
			setupCtx: func() context.Context {
				return t.Context()
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			version := GetBeatVersion(ctx)

			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestGetBeatName(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "beat name exists",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatNameCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "filebeat",
		},
		{
			name: "beat name missing",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "",
		},
		{
			name: "no client info",
			setupCtx: func() context.Context {
				return t.Context()
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			name := GetBeatName(ctx)
			assert.Equal(t, tt.expected, name)
		})
	}
}
