// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package otelctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/client"
)

func TestGetBeatEventMeta(t *testing.T) {
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
						BeatNameCtxKey:        {"filebeat"},
						BeatIndexPrefixCtxKey: {"filebeat"},
						BeatVersionCtxKey:     {"8.0.0"},
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
						BeatIndexPrefixCtxKey: {"filebeat"},
						BeatVersionCtxKey:     {"8.0.0"},
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
			name: "missing beat index prefix",
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
						BeatIndexPrefixCtxKey: {"filebeat"},
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

func TestGetBeatIndexPrefix(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "prefix exists",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						BeatIndexPrefixCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "filebeat",
		},
		{
			name: "prefix missing",
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
			name := GetBeatIndexPrefix(ctx)
			assert.Equal(t, tt.expected, name)
		})
	}
}
