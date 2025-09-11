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

	"go.opentelemetry.io/collector/client"

	"github.com/elastic/beats/v7/libbeat/beat"
)

const (
	BeatNameCtxKey    = "beat_name"
	BeatVersionCtxKey = "beat_version"
)

// NewConsumerContext creates a new context.Context adding the beats metadata
// to the client.Info. This is used to pass the beat name and version to the
// Collector, so it can be used by the components to access that data.
func NewConsumerContext(ctx context.Context, beatInfo beat.Info) context.Context {
	clientInfo := client.Info{
		Metadata: client.NewMetadata(map[string][]string{
			BeatNameCtxKey:    {beatInfo.Beat},
			BeatVersionCtxKey: {beatInfo.Version},
		}),
	}
	return client.NewContext(ctx, clientInfo)
}

// GetBeatName retrieves the beat name from the context metadata
// If the name is not found, it returns an empty string.
func GetBeatName(ctx context.Context) string {
	clientInfo := client.FromContext(ctx)
	if values := clientInfo.Metadata.Get(BeatNameCtxKey); len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetBeatVersion retrieves the version of the beat from the context metadata
// If the version is not found, it returns an empty string.
func GetBeatVersion(ctx context.Context) string {
	clientInfo := client.FromContext(ctx)
	if values := clientInfo.Metadata.Get(BeatVersionCtxKey); len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetBeatEventMeta gives beat.Event.Meta from the context metadata
func GetBeatEventMeta(ctx context.Context) map[string]any {
	ctxData := client.FromContext(ctx)
	var beatName, beatVersion string
	if v := ctxData.Metadata.Get(BeatNameCtxKey); len(v) > 0 {
		beatName = v[0]
	}
	if v := ctxData.Metadata.Get(BeatVersionCtxKey); len(v) > 0 {
		beatVersion = v[0]
	}
	return map[string]any{
		"beat":    beatName,
		"version": beatVersion,
	}
}
