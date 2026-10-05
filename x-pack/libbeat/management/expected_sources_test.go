// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func TestShareExpectedSources(t *testing.T) {
	source, err := structpb.NewStruct(map[string]any{
		"id":          "synthetics/http-1",
		"meta":        map[string]any{"package": map[string]any{"name": "synthetics"}},
		"data_stream": map[string]any{"namespace": "default"},
		"streams": []any{
			map[string]any{"id": "s1", "urls": "http://a", "data_stream": map[string]any{"dataset": "http"}},
		},
	})
	require.NoError(t, err)
	fields := source.GetFields()
	stream := fields["streams"].GetListValue().GetValues()[0].GetStructValue()
	built := &proto.UnitExpectedConfig{
		Source:     source,
		Meta:       &proto.Meta{Source: fields["meta"].GetStructValue(), Package: &proto.Package{Source: fields["meta"].GetStructValue().GetFields()["package"].GetStructValue()}},
		DataStream: &proto.DataStream{Source: fields["data_stream"].GetStructValue()},
		Streams:    []*proto.Stream{{Id: "s1", Source: stream, DataStream: &proto.DataStream{Source: stream.GetFields()["data_stream"].GetStructValue()}}},
	}

	// Simulate the wire: sharing is lost and every sub-message becomes its own copy.
	raw, err := gproto.Marshal(built)
	require.NoError(t, err)
	received := &proto.UnitExpectedConfig{}
	require.NoError(t, gproto.Unmarshal(raw, received))
	before := gproto.Clone(received)

	shareExpectedSources(received)

	receivedFields := received.GetSource().GetFields()
	receivedStream := receivedFields["streams"].GetListValue().GetValues()[0].GetStructValue()
	assert.Same(t, receivedStream, received.Streams[0].Source)
	assert.Same(t, receivedStream.GetFields()["data_stream"].GetStructValue(), received.Streams[0].DataStream.Source)
	assert.Same(t, receivedFields["meta"].GetStructValue(), received.Meta.Source)
	assert.Same(t, receivedFields["data_stream"].GetStructValue(), received.DataStream.Source)
	assert.True(t, gproto.Equal(before, received), "content must not change")
}

func TestShareExpectedSourcesKeepsDifferingContent(t *testing.T) {
	source, err := structpb.NewStruct(map[string]any{"streams": []any{map[string]any{"id": "s1"}}})
	require.NoError(t, err)
	other, err := structpb.NewStruct(map[string]any{"id": "different"})
	require.NoError(t, err)
	cfg := &proto.UnitExpectedConfig{Source: source, Streams: []*proto.Stream{{Source: other}}}

	shareExpectedSources(cfg)

	assert.Same(t, other, cfg.Streams[0].Source)
}

func TestEqualIgnoringPolicy(t *testing.T) {
	newConfig := func(revision float64, urls string) *proto.UnitExpectedConfig {
		source, err := structpb.NewStruct(map[string]any{
			"id":     "synthetics/http-1",
			"policy": map[string]any{"revision": revision},
			"streams": []any{
				map[string]any{"id": "s1", "urls": urls},
			},
		})
		require.NoError(t, err)
		stream := source.GetFields()["streams"].GetListValue().GetValues()[0].GetStructValue()
		return &proto.UnitExpectedConfig{Id: "synthetics/http-1", Type: "synthetics/http", Source: source, Streams: []*proto.Stream{{Id: "s1", Source: stream}}}
	}

	assert.True(t, equalIgnoringPolicy(newConfig(1, "http://a"), newConfig(1, "http://a")))
	assert.True(t, equalIgnoringPolicy(newConfig(1, "http://a"), newConfig(2, "http://a")), "policy revision must not matter")
	assert.False(t, equalIgnoringPolicy(newConfig(1, "http://a"), newConfig(1, "http://b")), "changed stream must matter")

	withoutPolicy := newConfig(1, "http://a")
	delete(withoutPolicy.Source.Fields, policySourceKey)
	assert.True(t, equalIgnoringPolicy(newConfig(1, "http://a"), withoutPolicy))

	otherKey := newConfig(1, "http://a")
	otherKey.Source.Fields["extra"] = structpb.NewStringValue("x")
	assert.False(t, equalIgnoringPolicy(newConfig(1, "http://a"), otherKey), "other source keys must matter")

	renamed := newConfig(1, "http://a")
	renamed.Type = "synthetics/tcp"
	assert.False(t, equalIgnoringPolicy(newConfig(1, "http://a"), renamed), "fields outside source must matter")

	assert.False(t, equalIgnoringPolicy(newConfig(1, "http://a"), nil))
}

// equalIgnoringPolicy compares fields explicitly; this fails when the client library adds one.
func TestEqualIgnoringPolicyCoversAllFields(t *testing.T) {
	count := func(m gproto.Message) int { return m.ProtoReflect().Descriptor().Fields().Len() }
	assert.Equal(t, 8, count(&proto.UnitExpectedConfig{}), "update equalIgnoringPolicy for the new UnitExpectedConfig field")
	assert.Equal(t, 3, count(&proto.Stream{}), "update equalIgnoringPolicy for the new Stream field")
	assert.Equal(t, 4, count(&proto.DataStream{}), "update equalIgnoringPolicy for the new DataStream field")
	assert.Equal(t, 2, count(&proto.Meta{}), "update equalIgnoringPolicy for the new Meta field")
	assert.Equal(t, 3, count(&proto.Package{}), "update equalIgnoringPolicy for the new Package field")
}

// equalIgnoringPolicy must agree with proto.Equal on everything except the policy key.
func TestEqualIgnoringPolicyAgreesWithProtoEqual(t *testing.T) {
	base := func() *proto.UnitExpectedConfig {
		source, err := structpb.NewStruct(map[string]any{
			"id": "u", "num": 1.5, "flag": true, "nothing": nil,
			"meta":        map[string]any{"package": map[string]any{"name": "synthetics"}},
			"data_stream": map[string]any{"namespace": "default"},
			"streams":     []any{map[string]any{"id": "s1", "list": []any{"a", 2.0, map[string]any{"k": "v"}}, "data_stream": map[string]any{"dataset": "http"}}},
		})
		require.NoError(t, err)
		f := source.GetFields()
		stream := f["streams"].GetListValue().GetValues()[0].GetStructValue()
		return &proto.UnitExpectedConfig{
			Id: "u", Type: "t", Name: "n", Revision: 3, Source: source,
			Meta:       &proto.Meta{Source: f["meta"].GetStructValue(), Package: &proto.Package{Name: "synthetics", Version: "1", Source: f["meta"].GetStructValue().GetFields()["package"].GetStructValue()}},
			DataStream: &proto.DataStream{Dataset: "d", Type: "t", Namespace: "default", Source: f["data_stream"].GetStructValue()},
			Streams:    []*proto.Stream{{Id: "s1", Source: stream, DataStream: &proto.DataStream{Dataset: "http", Source: stream.GetFields()["data_stream"].GetStructValue()}}},
		}
	}
	mutations := map[string]func(*proto.UnitExpectedConfig){
		"id":            func(c *proto.UnitExpectedConfig) { c.Id = "x" },
		"type":          func(c *proto.UnitExpectedConfig) { c.Type = "x" },
		"name":          func(c *proto.UnitExpectedConfig) { c.Name = "x" },
		"revision":      func(c *proto.UnitExpectedConfig) { c.Revision = 9 },
		"source number": func(c *proto.UnitExpectedConfig) { c.Source.Fields["num"] = structpb.NewNumberValue(2) },
		"source bool":   func(c *proto.UnitExpectedConfig) { c.Source.Fields["flag"] = structpb.NewBoolValue(false) },
		"source null":   func(c *proto.UnitExpectedConfig) { c.Source.Fields["nothing"] = structpb.NewStringValue("") },
		"stream id":     func(c *proto.UnitExpectedConfig) { c.Streams[0].Id = "x" },
		"stream list": func(c *proto.UnitExpectedConfig) {
			c.Streams[0].Source.Fields["list"].GetListValue().Values[2] = structpb.NewStringValue("x")
		},
		"stream data_stream": func(c *proto.UnitExpectedConfig) { c.Streams[0].DataStream.Dataset = "x" },
		"stream removed":     func(c *proto.UnitExpectedConfig) { c.Streams = nil },
		"unit data_stream":   func(c *proto.UnitExpectedConfig) { c.DataStream.Namespace = "x" },
		"meta package":       func(c *proto.UnitExpectedConfig) { c.Meta.Package.Version = "2" },
		"meta source":        func(c *proto.UnitExpectedConfig) { c.Meta.Source = structpb.NewStringValue("x").GetStructValue() },
		"meta removed":       func(c *proto.UnitExpectedConfig) { c.Meta = nil },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := base()
			mutate(changed)
			require.False(t, gproto.Equal(base(), changed), "mutation must change the message")
			assert.False(t, equalIgnoringPolicy(base(), changed))
		})
	}
	assert.True(t, equalIgnoringPolicy(base(), base()))
	assert.True(t, gproto.Equal(base(), base()))
}
