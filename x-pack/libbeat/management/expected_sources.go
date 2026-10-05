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

// policySourceKey is the key of the unit's Source holding the agent policy metadata
// (currently only the revision). It changes for every unit on every policy revision,
// and the beat config generated from a unit does not depend on it.
const policySourceKey = "policy"

// equalIgnoringPolicy reports whether a and b are equal, disregarding the policy
// metadata in their Source, so the beat configs generated from them are interchangeable.
//
// The comparison is explicit instead of using proto.Equal: it runs for every unit on every
// change, and proto.Equal's reflection allocated about as much as it saved. Fields added to
// these messages must be added here; TestEqualIgnoringPolicyCoversAllFields enforces that.
func equalIgnoringPolicy(a, b *proto.UnitExpectedConfig) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Id != b.Id || a.Type != b.Type || a.Name != b.Name || a.Revision != b.Revision ||
		len(a.Streams) != len(b.Streams) || !equalStructs(a.Source, b.Source, policySourceKey) ||
		!equalDataStreams(a.DataStream, b.DataStream) || !equalMeta(a.Meta, b.Meta) {
		return false
	}
	for i, sa := range a.Streams {
		sb := b.Streams[i]
		if sa == sb {
			continue
		}
		if sa == nil || sb == nil || sa.Id != sb.Id || !equalStructs(sa.Source, sb.Source, "") ||
			!equalDataStreams(sa.DataStream, sb.DataStream) {
			return false
		}
	}
	return true
}

func equalDataStreams(a, b *proto.DataStream) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Dataset == b.Dataset && a.Type == b.Type && a.Namespace == b.Namespace && equalStructs(a.Source, b.Source, "")
}

func equalMeta(a, b *proto.Meta) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil || !equalStructs(a.Source, b.Source, "") {
		return false
	}
	pa, pb := a.Package, b.Package
	if pa == pb {
		return true
	}
	if pa == nil || pb == nil {
		return false
	}
	return pa.Name == pb.Name && pa.Version == pb.Version && equalStructs(pa.Source, pb.Source, "")
}

// equalStructs compares two structs, ignoring the top-level key skip when it is not empty.
func equalStructs(a, b *structpb.Struct, skip string) bool {
	if a == b {
		return true
	}
	fa, fb := a.GetFields(), b.GetFields()
	na, nb := len(fa), len(fb)
	if _, ok := fa[skip]; ok {
		na--
	}
	if _, ok := fb[skip]; ok {
		nb--
	}
	if na != nb {
		return false
	}
	for k, va := range fa {
		if k == skip {
			continue
		}
		vb, ok := fb[k]
		if !ok || !equalValues(va, vb) {
			return false
		}
	}
	return true
}

func equalValues(a, b *structpb.Value) bool {
	if a == b {
		return true
	}
	switch av := a.GetKind().(type) {
	case *structpb.Value_StructValue:
		bv, ok := b.GetKind().(*structpb.Value_StructValue)
		return ok && equalStructs(av.StructValue, bv.StructValue, "")
	case *structpb.Value_ListValue:
		bv, ok := b.GetKind().(*structpb.Value_ListValue)
		if !ok || len(av.ListValue.GetValues()) != len(bv.ListValue.GetValues()) {
			return false
		}
		for i, v := range av.ListValue.GetValues() {
			if !equalValues(v, bv.ListValue.GetValues()[i]) {
				return false
			}
		}
		return true
	case *structpb.Value_StringValue:
		bv, ok := b.GetKind().(*structpb.Value_StringValue)
		return ok && av.StringValue == bv.StringValue
	case *structpb.Value_NumberValue:
		bv, ok := b.GetKind().(*structpb.Value_NumberValue)
		return ok && av.NumberValue == bv.NumberValue
	case *structpb.Value_BoolValue:
		bv, ok := b.GetKind().(*structpb.Value_BoolValue)
		return ok && av.BoolValue == bv.BoolValue
	case *structpb.Value_NullValue:
		_, ok := b.GetKind().(*structpb.Value_NullValue)
		return ok
	}
	return b.GetKind() == nil
}
