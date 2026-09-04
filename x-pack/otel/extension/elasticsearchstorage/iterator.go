// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
)

// defaultPageSize is the number of documents fetched per Walk page. Cursor
// stores are tiny, so a single page usually suffices; pagination via PIT +
// search_after handles the rare larger store. Tests shrink the client's
// pageSize field to exercise multi-page pagination cheaply.
const defaultPageSize = 1000

var _ storage.Walker = (*esStorageClient)(nil)

// Walk implements [storage.Walker]: it calls fn for every key/value pair in
// the store, paginating with a point-in-time (PIT) reader plus search_after,
// so it is not bounded by a single search page.
//
// Operations returned by fn are collected and applied in order once ranging
// finishes (or after fn returns [storage.SkipAll]). They are applied
// sequentially via the same per-op path as Batch, not transactionally. If fn
// returns any other error — or ranging itself fails — Walk stops and no
// collected operations are applied.
func (c *esStorageClient) Walk(ctx context.Context, fn storage.WalkFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.checkOpen(); err != nil {
		return err
	}

	// clientMu is held only per ES request (inside c.request): each page is
	// decoded into fresh slices under the lock, then fn runs without it, so
	// a long enumeration does not starve other users of the connection.

	pitID, ok, err := c.openPIT(ctx)
	if err != nil {
		return err
	}
	if !ok {
		// Index does not exist yet (no writes): nothing to enumerate.
		return nil
	}
	defer func() { _ = c.closePIT(pitID) }()

	var (
		ops         []*storage.Operation
		searchAfter []json.RawMessage
	)
	for {
		entries, nextAfter, newPIT, err := c.searchPage(ctx, pitID, searchAfter)
		if err != nil {
			return err
		}
		if newPIT != "" {
			pitID = newPIT
		}
		for _, e := range entries {
			got, ferr := fn(e.key, e.value)
			if ferr != nil {
				if errors.Is(ferr, storage.SkipAll) {
					// Stop ranging but still apply what fn collected.
					return c.Batch(ctx, append(ops, got...)...)
				}
				// Abort without applying any collected operations.
				return ferr
			}
			ops = append(ops, got...)
		}
		if len(entries) < c.pageSize {
			return c.Batch(ctx, ops...)
		}
		searchAfter = nextAfter
	}
}

// searchEntry is a decoded hit: the original (unescaped) key, its decoded
// value, and the ES sort cursor used for search_after pagination.
type searchEntry struct {
	key   string
	value []byte
	sort  []json.RawMessage
}

// openPIT opens a point-in-time reader over the store index. ok is false (with
// a nil error) when the index does not exist yet, meaning there is nothing to
// enumerate.
func (c *esStorageClient) openPIT(ctx context.Context) (id string, ok bool, err error) {
	status, body, err := c.request(ctx, "POST", "/"+c.index+"/_pit", map[string]string{"keep_alive": "1m"}, nil)
	if status == http.StatusNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("opening point-in-time reader: %w", err)
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", false, fmt.Errorf("decoding point-in-time response: %w", err)
	}
	if resp.ID == "" {
		return "", false, fmt.Errorf("elasticsearch_storage: empty point-in-time id")
	}
	return resp.ID, true, nil
}

// closePIT releases a point-in-time reader. Errors are non-fatal (the PIT
// expires on its keep_alive anyway) and are returned for the caller to log.
func (c *esStorageClient) closePIT(id string) error {
	encoded, err := json.Marshal(map[string]string{"id": id})
	if err != nil {
		return err
	}
	_, _, err = c.request(context.Background(), "DELETE", "/_pit", nil, eslegclient.RawEncoding{Encoding: encoded})
	return err
}

// searchPage fetches one page of the enumeration. It returns the decoded
// entries, the sort cursor for the next page, and the (possibly refreshed) PIT
// id ES returns with each search response.
func (c *esStorageClient) searchPage(ctx context.Context, pitID string, after []json.RawMessage) (entries []searchEntry, nextAfter []json.RawMessage, newPIT string, err error) {
	query := map[string]any{
		"size":             c.pageSize,
		"track_total_hits": false,
		"query":            map[string]any{"match_all": map[string]any{}},
		"pit":              map[string]any{"id": pitID, "keep_alive": "1m"},
		// _shard_doc is the implicit, efficient tiebreak sort available only
		// with a PIT; it gives a stable total order for search_after.
		"sort": []any{map[string]any{"_shard_doc": "asc"}},
	}
	if len(after) > 0 {
		query["search_after"] = after
	}
	encoded, err := json.Marshal(query)
	if err != nil {
		return nil, nil, "", fmt.Errorf("encoding enumeration query: %w", err)
	}

	// Note: a PIT search targets _search with no index in the path.
	_, body, err := c.request(ctx, "POST", "/_search", nil, eslegclient.RawEncoding{Encoding: encoded})
	if err != nil {
		return nil, nil, "", fmt.Errorf("enumerating storage documents: %w", err)
	}

	var resp struct {
		PitID string `json:"pit_id"`
		Hits  struct {
			Hits []struct {
				ID     string            `json:"_id"`
				Source docBody           `json:"_source"`
				Sort   []json.RawMessage `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, "", fmt.Errorf("decoding enumeration response: %w", err)
	}

	entries = make([]searchEntry, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		key, uerr := url.PathUnescape(h.ID)
		if uerr != nil {
			// A document _id we did not write; skip rather than abort the
			// whole enumeration.
			key = h.ID
		}
		value, derr := decodeValue(h.Source.V, h.Source.Enc)
		if derr != nil {
			return nil, nil, "", fmt.Errorf("decoding value for key %q: %w", key, derr)
		}
		entries = append(entries, searchEntry{key: key, value: value, sort: h.Sort})
	}
	if n := len(entries); n > 0 {
		nextAfter = entries[n-1].sort
	}
	return entries, nextAfter, resp.PitID, nil
}
