// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

//go:build integration

package elasticsearchstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// newTestExtension builds and starts an elasticStorage configured against
// the integration ES cluster. Cleanup is registered to Shutdown the
// extension at end of test.
func newTestExtension(t *testing.T) *elasticStorage {
	t.Helper()

	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	cfg := &Config{
		ElasticsearchConfig: map[string]interface{}{
			"hosts":    []string{fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host)},
			"username": user,
			"password": pass,
		},
	}

	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, "")}
	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })
	return ext
}

// newTestClient returns an OTel storage.Client scoped to a per-test
// component ID derived from t.Name(), so concurrent test runs don't
// share an ES index.
func newTestClient(t *testing.T, ext *elasticStorage) storage.Client {
	t.Helper()
	// Sanitize the test name down to component-ID-legal characters
	// (alphanumeric + underscore). t.Name() can contain '/'.
	name := strings.NewReplacer("/", "_", "-", "_", " ", "_").Replace(strings.ToLower(t.Name()))
	id := component.MustNewIDWithName("test_storage", name)
	client, err := ext.GetClient(context.Background(), component.KindReceiver, id, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })
	return client
}

func TestESClient_GetMissing(t *testing.T) {
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	got, err := c.Get(context.Background(), "no-such-key")
	require.NoError(t, err)
	assert.Nil(t, got, "Get on a missing key must return (nil, nil) per the OTel contract")
}

func TestESClient_SetGetRoundTrip_Struct(t *testing.T) {
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	type cursor struct {
		Offset    string `json:"offset"`
		Timestamp int64  `json:"timestamp"`
		CaughtUp  bool   `json:"caught_up"`
	}
	want := cursor{Offset: "abc123", Timestamp: 1776145950, CaughtUp: false}
	encoded, err := json.Marshal(want)
	require.NoError(t, err)

	require.NoError(t, c.Set(context.Background(), "cursor", encoded))

	gotBytes, err := c.Get(context.Background(), "cursor")
	require.NoError(t, err)
	require.NotNil(t, gotBytes)

	var got cursor
	require.NoError(t, json.Unmarshal(gotBytes, &got))
	assert.Equal(t, want, got)
}

func TestESClient_SetGetRoundTrip_LargeInt(t *testing.T) {
	// Values exceeding 2^53 lose precision when round-tripped through
	// float64. The adapter embeds the caller's bytes verbatim under `v`
	// (mapped object/enabled:false), so ES preserves them as-is.
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	const large int64 = 9_000_000_000_000_000_001 // > 2^53
	in := []byte(fmt.Sprintf(`{"big":%d}`, large))
	require.NoError(t, c.Set(context.Background(), "big", in))

	out, err := c.Get(context.Background(), "big")
	require.NoError(t, err)

	// json.Number preserves the exact decimal representation; compare
	// the integer value to confirm no precision was lost.
	dec := json.NewDecoder(strings.NewReader(string(out)))
	dec.UseNumber()
	var decoded map[string]json.Number
	require.NoError(t, dec.Decode(&decoded))

	got, err := decoded["big"].Int64()
	require.NoError(t, err)
	assert.Equal(t, large, got)
}

func TestESClient_SetGetRoundTrip_RawBytes(t *testing.T) {
	// Non-JSON bytes are accepted and round-trip byte-for-byte via the
	// base64 envelope (enc:base64). This is the generality the OTel
	// storage.Client contract requires (protobuf/CBOR/compressed blobs).
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	in := []byte{0x00, 0x01, 0x02, 0xff, 0xfe}
	require.NoError(t, c.Set(context.Background(), "binary", in))

	got, err := c.Get(context.Background(), "binary")
	require.NoError(t, err)
	assert.Equal(t, in, got, "non-JSON bytes must round-trip exactly")
}

func TestESClient_SetGetRoundTrip_BareJSONValue(t *testing.T) {
	// A value that is a bare JSON number/string is valid JSON, so it takes
	// the verbatim json branch and must come back byte-identical.
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	for _, in := range [][]byte{[]byte(`42`), []byte(`"hello"`), []byte(`true`)} {
		require.NoError(t, c.Set(context.Background(), "bare", in))
		got, err := c.Get(context.Background(), "bare")
		require.NoError(t, err)
		assert.Equal(t, in, got, "bare JSON value must round-trip exactly")
	}
}

func TestESClient_EmptyKey_Rejected_NoRequest(t *testing.T) {
	// Empty keys are rejected up front (errEmptyKey) and must not issue any
	// request to ES.
	ext := newTestExtension(t)
	c := newTestClient(t, ext)
	ctx := context.Background()

	_, err := c.Get(ctx, "")
	assert.ErrorIs(t, err, errEmptyKey)
	assert.ErrorIs(t, c.Set(ctx, "", []byte(`{}`)), errEmptyKey)
	assert.ErrorIs(t, c.Delete(ctx, ""), errEmptyKey)

	// A batch containing an empty-key op fails the batch with errEmptyKey.
	err = c.Batch(ctx, storage.SetOperation("", []byte(`{}`)))
	assert.ErrorIs(t, err, errEmptyKey)
}

func TestESClient_Delete(t *testing.T) {
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	require.NoError(t, c.Set(context.Background(), "k", []byte(`{"a":1}`)))
	require.NoError(t, c.Delete(context.Background(), "k"))

	got, err := c.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestESClient_DeleteMissing_NoOp(t *testing.T) {
	// OTel contract: "Delete doesn't error if the key doesn't exist - it
	// just no-ops."
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	assert.NoError(t, c.Delete(context.Background(), "never-existed"))
}

func TestESClient_CloseIdempotent(t *testing.T) {
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	require.NoError(t, c.Close(context.Background()))
	assert.NoError(t, c.Close(context.Background()))
}

func TestESClient_OpAfterClose(t *testing.T) {
	ext := newTestExtension(t)
	c := newTestClient(t, ext)

	require.NoError(t, c.Close(context.Background()))

	_, err := c.Get(context.Background(), "k")
	assert.ErrorIs(t, err, errClientClosed)

	err = c.Set(context.Background(), "k", []byte(`{}`))
	assert.ErrorIs(t, err, errClientClosed)

	err = c.Delete(context.Background(), "k")
	assert.ErrorIs(t, err, errClientClosed)
}

func TestESClient_Batch(t *testing.T) {
	ext := newTestExtension(t)
	c := newTestClient(t, ext)
	ctx := context.Background()

	// Batch with mixed Set/Get/Delete. The PR 1 implementation runs ops
	// sequentially; we just verify each one took effect.
	err := c.Batch(ctx,
		storage.SetOperation("a", []byte(`{"v":1}`)),
		storage.SetOperation("b", []byte(`{"v":2}`)),
		storage.GetOperation("a"),
		storage.DeleteOperation("b"),
		storage.GetOperation("b"),
	)
	require.NoError(t, err)

	// Re-issue the gets to verify state.
	got, err := c.Get(ctx, "a")
	require.NoError(t, err)
	assert.JSONEq(t, `{"v":1}`, string(got))

	got, err = c.Get(ctx, "b")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetClient_NamedReceivers_DistinctIndices(t *testing.T) {
	// Reproduces the elastic/beats#50223 scenario: two receivers of the
	// same type but different names must each get their own valid ES
	// index, not collide and not break with an invalid index name.
	ext := newTestExtension(t)

	idRaw := component.MustNewIDWithName("akamai_siem", "raw")
	idOtel := component.MustNewIDWithName("akamai_siem", "otel")

	cRaw, err := ext.GetClient(context.Background(), component.KindReceiver, idRaw, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = cRaw.Close(context.Background()) })

	cOtel, err := ext.GetClient(context.Background(), component.KindReceiver, idOtel, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = cOtel.Close(context.Background()) })

	require.NoError(t, cRaw.Set(context.Background(), "k", []byte(`{"who":"raw"}`)))
	require.NoError(t, cOtel.Set(context.Background(), "k", []byte(`{"who":"otel"}`)))

	rawGot, err := cRaw.Get(context.Background(), "k")
	require.NoError(t, err)
	otelGot, err := cOtel.Get(context.Background(), "k")
	require.NoError(t, err)

	assert.JSONEq(t, `{"who":"raw"}`, string(rawGot))
	assert.JSONEq(t, `{"who":"otel"}`, string(otelGot))
}

func TestGetClient_LazyIndexCreation(t *testing.T) {
	// GetClient must not create the index (creation is deferred to the first
	// write), so acquiring a client never fails on index setup. A second
	// write is a no-op create ("resource_already_exists_exception" tolerated),
	// and two clients for the same identity coexist.
	ext := newTestExtension(t)
	ctx := context.Background()

	id := component.MustNewIDWithName("test_storage", "lazy_index")
	c1, err := ext.GetClient(ctx, component.KindReceiver, id, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c1.Close(ctx) })

	c2, err := ext.GetClient(ctx, component.KindReceiver, id, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c2.Close(ctx) })

	// First write creates the index; subsequent writes are no-op creates.
	require.NoError(t, c1.Set(ctx, "k", []byte(`{"a":1}`)))
	require.NoError(t, c1.Set(ctx, "k", []byte(`{"a":2}`)))
	require.NoError(t, c2.Set(ctx, "k2", []byte(`{"b":1}`)))
}

func TestGetClient_MultiReceiver_Concurrent(t *testing.T) {
	// N clients with distinct identities writing and reading in parallel must
	// not race (run under -race) and must not cross-contaminate: each reads
	// back exactly its own value.
	ext := newTestExtension(t)
	ctx := context.Background()

	const numClients = 8
	clients := make([]storage.Client, numClients)
	for i := 0; i < numClients; i++ {
		id := component.MustNewIDWithName("test_storage", fmt.Sprintf("mr_%d", i))
		c, err := ext.GetClient(ctx, component.KindReceiver, id, "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.Close(ctx) })
		clients[i] = c
	}

	var wg sync.WaitGroup
	errs := make([]error, numClients)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			val := []byte(fmt.Sprintf(`{"who":%d}`, i))
			for j := 0; j < 20; j++ {
				if err := clients[i].Set(ctx, "k", val); err != nil {
					errs[i] = err
					return
				}
				got, err := clients[i].Get(ctx, "k")
				if err != nil {
					errs[i] = err
					return
				}
				if string(got) != string(val) {
					errs[i] = fmt.Errorf("client %d read %q, want %q", i, got, val)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "client %d", i)
	}
}

func TestESClient_SchemaEvolution(t *testing.T) {
	// The index mapping declares `v` as object/enabled:false, so ES
	// stores arbitrary JSON shapes under it without parsing or
	// type-checking. Two writes with structurally different `v` shapes
	// must both succeed.
	ext := newTestExtension(t)
	c := newTestClient(t, ext)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", []byte(`{"shape":"a","count":5}`)))
	// Different field types and shape — would fail under dynamic mapping.
	require.NoError(t, c.Set(ctx, "k", []byte(`{"shape":42,"items":[1,2,3]}`)))

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.JSONEq(t, `{"shape":42,"items":[1,2,3]}`, string(got))
}
