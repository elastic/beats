// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	gproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// shareExpectedSources points the Source of the typed sub-messages (streams, meta,
// data_stream) at the matching parts of the unit's Source. The agent builds them that
// way, but the wire format cannot express sharing, so every stream config arrives and is
// kept twice. With one unit per Synthetics monitor that duplicate is a large part of
// heartbeat's memory. The client only replaces a unit's config when its index changes,
// so rewriting the pointers in place is safe.
func shareExpectedSources(cfg *proto.UnitExpectedConfig) {
	source := cfg.GetSource()
	if source == nil {
		return
	}
	fields := source.GetFields()

	if cfg.Meta != nil {
		if meta := fields["meta"].GetStructValue(); meta != nil {
			cfg.Meta.Source = shared(cfg.Meta.Source, meta)
			if cfg.Meta.Package != nil {
				cfg.Meta.Package.Source = shared(cfg.Meta.Package.Source, meta.GetFields()["package"].GetStructValue())
			}
		}
	}
	if cfg.DataStream != nil {
		cfg.DataStream.Source = shared(cfg.DataStream.Source, fields["data_stream"].GetStructValue())
	}

	streams := fields["streams"].GetListValue().GetValues()
	if len(streams) != len(cfg.Streams) {
		return
	}
	for i, stream := range cfg.Streams {
		streamSource := streams[i].GetStructValue()
		stream.Source = shared(stream.Source, streamSource)
		if stream.DataStream != nil && streamSource != nil {
			stream.DataStream.Source = shared(stream.DataStream.Source, streamSource.GetFields()["data_stream"].GetStructValue())
		}
	}
}

// shared returns candidate when it holds the same content as current, so current can be
// garbage collected; otherwise current is kept.
func shared(current, candidate *structpb.Struct) *structpb.Struct {
	if current == nil || candidate == nil || current == candidate {
		return current
	}
	if !gproto.Equal(current, candidate) {
		return current
	}
	return candidate
}
