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
