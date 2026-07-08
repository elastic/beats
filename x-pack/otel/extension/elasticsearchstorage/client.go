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

var (
	// errClientClosed is returned by Get/Set/Delete/Batch/Each after Close
	// has been called on the client. Callers can use errors.Is to detect it.
	errClientClosed = errors.New("elasticsearch_storage: client is closed")

	// errEmptyKey is returned when a keyed operation is given an empty key.
	// An empty key would collapse the document path to "/<index>/_doc/",
	// which behaves differently per HTTP verb and can strand documents; we
	// reject it up front instead.
	errEmptyKey = errors.New("elasticsearch_storage: key must not be empty")
)

// esStorageClient is the OTel storage.Client implementation backed by the
// single shared *eslegclient.Connection owned by the parent extension. Every
// method that touches the connection takes the extension's clientMu:
// eslegclient.Connection is not safe for concurrent use (it reuses an
// internal response buffer and a body encoder), so serializing on clientMu is
// what keeps the OTel path, the Access path, and the entcollect path from
// corrupting each other.
type esStorageClient struct {
	ext   *elasticStorage
	index string

	// ensureMu guards lazy, idempotent index creation. Only success is
	// cached, so a transient failure on the first write is retried on the
	// next one rather than permanently disabling the client.
	ensureMu sync.Mutex
	ensured  bool

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
	if err := checkKey(key); err != nil {
		return nil, err
	}

	status, body, err := c.request(ctx, "GET", c.docPath(key), nil, nil)
	// Check status before err: eslegclient returns a non-nil error for any
	// status >= 300, including the expected 404 for a missing key.
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
	return decodeValue(resp.Source.V, resp.Source.Enc)
}

// Set stores value under key. Arbitrary bytes are accepted: valid JSON is
// stored verbatim under `v` (enc:json, readable in Kibana, precision and key
// order preserved); anything else is base64-wrapped (enc:base64). The
// encoding mode is configurable (see Config.Encoding).
func (c *esStorageClient) Set(ctx context.Context, key string, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.checkOpen(); err != nil {
		return err
	}
	if err := checkKey(key); err != nil {
		return err
	}
	if err := c.ensureIndex(); err != nil {
		return err
	}

	raw, enc := encodeValue(value, c.encoding())
	encoded, err := json.Marshal(docBody{
		V:         raw,
		Enc:       enc,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("encoding storage document: %w", err)
	}

	// RawEncoding tells the eslegclient encoder to use the bytes as-is
	// rather than re-serializing through go-structform; this preserves the
	// verbatim `v` payload.
	_, _, err = c.request(ctx, "PUT", c.docPath(key), c.writeParams(), eslegclient.RawEncoding{Encoding: encoded})
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
	if err := checkKey(key); err != nil {
		return err
	}

	status, _, err := c.request(ctx, "DELETE", c.docPath(key), c.writeParams(), nil)
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
// This keeps the implementation a per-op loop — correct, but one HTTP
// round-trip per op. A future change may switch to ES _bulk for fewer
// round-trips. The OTel contract does not require Batch to be transactional,
// and ES offers no cross-document atomicity even via _bulk, so partial-failure
// semantics (partial state on error) are unchanged either way.
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

// Close marks the client closed. The shared *eslegclient.Connection is owned
// by the extension and torn down in Shutdown; closing a client is a
// "stop using me" signal local to that client, so one client closing cannot
// take down the shared connection or other clients.
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

// checkKey rejects empty keys before any path is built (see errEmptyKey).
func checkKey(key string) error {
	if key == "" {
		return errEmptyKey
	}
	return nil
}

// docPath returns the ES document path for key. url.PathEscape is used
// (not QueryEscape) because it is the technically correct escaping for a path
// segment; it diverges from the legacy baseStore (QueryEscape) only on '+'
// and space. The OTel client writes to its own indices, distinct from any
// baseStore index, so there is no cross-path key collision.
func (c *esStorageClient) docPath(key string) string {
	return fmt.Sprintf("/%s/_doc/%s", c.index, url.PathEscape(key))
}

// encoding returns the effective encoding mode, defaulting an empty config to
// "auto".
func (c *esStorageClient) encoding() string {
	if c.ext.cfg.Encoding == "" {
		return "auto"
	}
	return c.ext.cfg.Encoding
}

// writeParams returns the query parameters for write requests, carrying the
// configured refresh mode when set (nil otherwise, preserving the default
// no-force-refresh behaviour).
func (c *esStorageClient) writeParams() map[string]string {
	if c.ext.cfg.Refresh == "" {
		return nil
	}
	return map[string]string{"refresh": c.ext.cfg.Refresh}
}

// ensureIndex lazily creates the storage index on first write. It is
// idempotent (a concurrent creator's "resource_already_exists_exception" is
// treated as success) and caches only success, so a transient failure is
// retried on the next write rather than permanently disabling the client.
func (c *esStorageClient) ensureIndex() error {
	c.ensureMu.Lock()
	defer c.ensureMu.Unlock()
	if c.ensured {
		return nil
	}
	if err := ensureIndex(&c.ext.clientMu, c.ext.client, c.index, c.ext.cfg.Index); err != nil {
		return err
	}
	c.ensured = true
	return nil
}
