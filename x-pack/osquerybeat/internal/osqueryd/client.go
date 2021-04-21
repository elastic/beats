// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqueryd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/kolide/osquery-go"
)

const (
	DefaultTimeout = 30 * time.Second

	lruCacheSize = 1024

	retryWait  = 200 * time.Millisecond
	retryTimes = 10
	logTag     = "osqueryd_cli"
)

type Cache interface {
	Add(key, value interface{}) (evicted bool)
	Get(key interface{}) (value interface{}, ok bool)
	Resize(size int) (evicted int)
}

type Client struct {
	cli   *osquery.ExtensionManagerClient
	cache Cache
	log   *logp.Logger
}

type Option func(*Client)

func NewClient(ctx context.Context, socketPath string, to time.Duration, log *logp.Logger, opts ...Option) (*Client, error) {
	cli, err := newClientWithRetries(ctx, socketPath, to)
	if err != nil {
		return nil, err
	}
	c := &Client{
		cli: cli,
		log: log,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

func WithCache(cache Cache) Option {
	return func(c *Client) {
		c.cache = cache
	}
}

func newClientWithRetries(ctx context.Context, socketPath string, to time.Duration) (cli *osquery.ExtensionManagerClient, err error) {
	log := logp.NewLogger(logTag).With("socket_path", socketPath)
	for i := 0; i < retryTimes; i++ {
		attempt := i + 1
		llog := log.With("attempt", attempt)
		llog.Debug("Connecting")
		cli, err = osquery.NewClient(socketPath, to)
		if err != nil {
			llog.Debug("Failed to connect, err: %v", err)
			if i < retryTimes-1 {
				llog.Infof("Wait for %v before next connect attempt", retryWait)
				if werr := waitWithContext(ctx, retryWait); werr != nil {
					err = werr
					break // Context cancelled, exit loop
				}
			}
			continue
		}
		break
	}
	if err != nil {
		log.Error("Failed to connect, err: %v", err)
	} else {
		log.Info("Connected.")
	}
	return cli, err
}

func (c *Client) Close() {
	if c.cli != nil {
		c.cli.Close()
		c.cli = nil
	}
}

func (c *Client) Query(ctx context.Context, sql string) ([]map[string]interface{}, error) {
	res, err := c.cli.Client.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("osquery failed: %w", err)
	}
	if res.Status.Code != int32(0) {
		return nil, errors.New(res.Status.Message)
	}

	// Get column types
	colTypes, err := c.queryColumnTypes(ctx, sql)
	if err != nil {
		return nil, err
	}

	return resolveTypes(res.Response, colTypes), nil
}

func (c *Client) queryColumnTypes(ctx context.Context, sql string) (map[string]string, error) {
	var colTypes map[string]string
	if c.cache != nil {
		if v, ok := c.cache.Get(sql); ok {
			colTypes, ok = v.(map[string]string)
			if ok {
				c.log.Debug("Using cached column types for query: %s", sql)
			} else {
				c.log.Error("Failed get the column types from cache, incompatible type")
			}
		}
	}
	if colTypes == nil {
		exres, err := c.cli.GetQueryColumns(sql)
		if err != nil {
			return nil, fmt.Errorf("osquery get query columns failed: %w", err)
		}

		colTypes = make(map[string]string)
		for _, m := range exres.Response {
			for k, v := range m {
				colTypes[k] = v
			}
		}
		c.cache.Add(sql, colTypes)
	}
	return colTypes, nil
}

func resolveTypes(hits []map[string]string, colTypes map[string]string) []map[string]interface{} {
	resolved := make([]map[string]interface{}, 0, len(hits))
	for _, hit := range hits {
		res := resolveHitTypes(hit, colTypes)
		resolved = append(resolved, res)
	}
	return resolved
}

// Best effort to convert value types and replace values in the
// If conversion fails the value is kept as string
func resolveHitTypes(hit, colTypes map[string]string) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range hit {
		t, ok := colTypes[k]
		if ok {
			var err error
			switch t {
			case "BIGINT", "INTEGER":
				var n int64
				n, err = strconv.ParseInt(v, 10, 64)
				if err == nil {
					m[k] = n
				}
			case "UNSIGNED_BIGINT":
				var n uint64
				n, err = strconv.ParseUint(v, 10, 64)
				if err == nil {
					m[k] = n
				}
			case "DOUBLE":
				var n float64
				n, err = strconv.ParseFloat(v, 64)
				if err == nil {
					m[k] = n
				}
			default:
				m[k] = v
			}
		} else {
			m[k] = v
		}
	}
	return m
}

func waitWithContext(ctx context.Context, to time.Duration) error {
	t := time.NewTimer(to)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return context.Canceled
	case <-t.C:
	}
	return nil
}
