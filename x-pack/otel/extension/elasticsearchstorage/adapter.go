// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
)

// errClientClosed is returned by Get/Set/Delete/Batch after Close has been
// called on the client. Callers can use errors.Is to detect this state.
var errClientClosed = errors.New("elasticsearch_storage: client is closed")

// docBody is the wire format the adapter writes and reads. Field tags use
// encoding/json (not the eslegclient go-structform "struct" tag) because
// the body is marshaled with stdlib json and submitted via
// eslegclient.RawEncoding — that bypasses go-structform entirely. Going
// through stdlib lets us embed the caller's value verbatim as
// json.RawMessage, guaranteeing the bytes ES sees under `v` are the same
// bytes the caller passed to Set.
type docBody struct {
	V         json.RawMessage `json:"v"`
	UpdatedAt string          `json:"updated_at"`
}

// esStorageClient is the OTel storage.Client implementation backed by a
// single shared *eslegclient.Connection on the parent extension. All
// methods take the extension's mutex around the Connection call;
// eslegclient.Connection is not safe for concurrent use because it reuses
// an internal response buffer.
//
// One *eslegclient.Connection is shared across all clients (and the
// legacy backend.Registry path). Per @cmacknz on issue #50223, this scales
// better than per-receiver connections when an agent runs many receivers.
// At cursor-state op rates the mutex contention is negligible.
type esStorageClient struct {
	ext   *elasticStorage
	index string

	closedMu sync.Mutex
	closed   bool
}

var _ storage.Client = (*esStorageClient)(nil)

// Get returns the value for key, or (nil, nil) if the key does not exist —
// per the OTel storage.Client contract: "Get doesn't error if a key is not
// found - it just returns nil."
func (c *esStorageClient) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.checkOpen(); err != nil {
		return nil, err
	}

	c.ext.mu.Lock()
	defer c.ext.mu.Unlock()

	status, body, err := c.ext.client.Request(
		"GET",
		c.docPath(key),
		"", nil, nil,
	)
	if status == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading storage document: %w", err)
	}

	var resp struct {
		Source docBody `json:"_source"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding storage response: %w", err)
	}
	if len(resp.Source.V) == 0 {
		// Document exists but has no `v` (shouldn't happen via this adapter,
		// but a corrupt or externally-written doc shouldn't crash callers).
		return nil, nil
	}
	// json.Unmarshal allocates a fresh slice for json.RawMessage, so the
	// returned bytes are not aliased to conn.responseBuffer (which the
	// connection reuses on the next call).
	return []byte(resp.Source.V), nil
}

// Set stores value under key. The value must be valid JSON: it is embedded
// verbatim under `v` in the document, and ES is configured to store `v` as
// an opaque object (enabled: false), so the bytes round-trip exactly. A
// non-JSON input is rejected with a typed error rather than silently
// corrupted.
func (c *esStorageClient) Set(ctx context.Context, key string, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.checkOpen(); err != nil {
		return err
	}
	if !json.Valid(value) {
		return fmt.Errorf("elasticsearch_storage: value for key %q is not valid JSON", key)
	}

	encoded, err := json.Marshal(docBody{
		V:         value,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("encoding storage document: %w", err)
	}

	c.ext.mu.Lock()
	defer c.ext.mu.Unlock()

	// RawEncoding tells the eslegclient encoder to use the bytes as-is
	// rather than re-serializing through go-structform; this preserves
	// numeric precision and key order in the user's `v` value.
	_, _, err = c.ext.client.Request(
		"PUT",
		c.docPath(key),
		"", nil,
		eslegclient.RawEncoding{Encoding: encoded},
	)
	if err != nil {
		return fmt.Errorf("writing storage document: %w", err)
	}
	return nil
}

// Delete removes the value for key. Per OTel contract, deleting a missing
// key is a no-op (not an error).
func (c *esStorageClient) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.checkOpen(); err != nil {
		return err
	}

	c.ext.mu.Lock()
	defer c.ext.mu.Unlock()

	status, _, err := c.ext.client.Request(
		"DELETE",
		c.docPath(key),
		"", nil, nil,
	)
	if status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting storage document: %w", err)
	}
	return nil
}

// Batch executes the supplied operations sequentially.
//
// PR 1 keeps this as a per-op loop — correct, but one HTTP round-trip per
// op. A future PR will switch to ES `_bulk` for fewer round-trips. The
// OTel contract does not require Batch to be transactional, and ES does
// not offer cross-document atomicity even via `_bulk`, so partial-failure
// semantics are unchanged either way.
func (c *esStorageClient) Batch(ctx context.Context, ops ...*storage.Operation) error {
	for _, op := range ops {
		if op == nil {
			continue
		}
		var err error
		switch op.Type {
		case storage.Get:
			op.Value, err = c.Get(ctx, op.Key)
		case storage.Set:
			err = c.Set(ctx, op.Key, op.Value)
		case storage.Delete:
			err = c.Delete(ctx, op.Key)
		default:
			return fmt.Errorf("elasticsearch_storage: unknown batch op type %d", op.Type)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Close marks the client closed. The shared *eslegclient.Connection is
// owned by the extension and torn down in Shutdown; closing a client is
// a "stop using me" signal local to that client, so callers cannot
// accidentally take down the shared connection by closing one client.
func (c *esStorageClient) Close(_ context.Context) error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	c.closed = true
	return nil
}

func (c *esStorageClient) checkOpen() error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	if c.closed {
		return errClientClosed
	}
	return nil
}

// docPath returns the ES document path for key. url.PathEscape is used
// (not QueryEscape) so '+' and ' ' are encoded as %2B / %20 — the legacy
// baseStore uses QueryEscape, but PathEscape is the technically correct
// choice for path segments. A new key written via the new adapter will
// not conflict with an existing baseStore key because the indices are
// distinct.
func (c *esStorageClient) docPath(key string) string {
	return fmt.Sprintf("/%s/_doc/%s", c.index, url.PathEscape(key))
}
