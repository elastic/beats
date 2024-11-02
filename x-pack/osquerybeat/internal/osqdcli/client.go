// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/osquery/osquery-go"
	genosquery "github.com/osquery/osquery-go/gen/osquery"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// The default query timeout
	defaultTimeout = 1 * time.Minute

	// The longest the query is allowed to run. Since queries are run one at a time, this will block all other queries until this query completes.
	defaultMaxTimeout     = 15 * time.Minute
	defaultConnectRetries = 10
)

// Hardcoded retry values
const (
	retryWait = 200 * time.Millisecond
)

// Limit number of queries across the socket
const (
	limit = 1
)

var (
	ErrAlreadyConnected = errors.New("already connected")
	ErrClientClosed     = errors.New("client is closed")
)

type ErrorQueryFailure struct {
	code    int32
	message string
}

func (e *ErrorQueryFailure) Error() string {
	return fmt.Sprintf("query failed, code: %d, message: %s", e.code, e.message)
}

type Client struct {
	socketPath string

	// Query timeout, currently can only be set at the transport level.
	// This means that while the query will return with error the osqueryd internally continues to execute the query until completion.
	// This is a known issue with osquery/osquery-go/thrift RPC implementation at the moment: there is effectively no way to cancel the long running query
	timeout        time.Duration
	maxTimeout     time.Duration
	connectRetries int

	log *logp.Logger

	cli *osquery.ExtensionManagerClient
	mx  sync.Mutex

	cache Cache

	cliLimiter *semaphore.Weighted
}

type Option func(*Client)

func WithTimeout(to time.Duration) Option {
	return func(c *Client) {
		c.timeout = to
	}
}

// WithMaxTimeout allows to define the max timeout per query, default is defaultMaxTimeout
func WithMaxTimeout(maxTimeout time.Duration) Option {
	return func(c *Client) {
		c.maxTimeout = maxTimeout
	}
}

func WithLogger(log *logp.Logger) Option {
	return func(c *Client) {
		c.log = log
	}
}

func WithConnectRetries(retries int) Option {
	return func(c *Client) {
		c.connectRetries = retries
	}
}

func New(socketPath string, opts ...Option) *Client {
	c := &Client{
		socketPath:     socketPath,
		timeout:        defaultTimeout,
		maxTimeout:     defaultMaxTimeout,
		connectRetries: defaultConnectRetries,
		cache:          &nullSafeCache{},
		cliLimiter:     semaphore.NewWeighted(limit),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) Connect(ctx context.Context) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.log.Debugf("connect osquery client: socket_path: %s, retries: %v", c.socketPath, c.connectRetries)
	if c.cli != nil {
		err := ErrAlreadyConnected
		c.log.Error(err)
		return err
	}

	err := c.reconnect(ctx)
	if err != nil {
		c.log.Errorf("osquery client failed to connect: %v", err)
		return err
	}
	c.log.Info("osquery client is connected")
	return err
}

func (c *Client) reconnect(ctx context.Context) error {
	c.close()
	cli, err := c.connectWithRetry(ctx, c.timeout)
	if err != nil {
		return err
	}
	c.cli = cli
	return nil
}

func (c *Client) connectWithRetry(ctx context.Context, timeout time.Duration) (cli *osquery.ExtensionManagerClient, err error) {
	r := retry{
		maxRetry:  c.connectRetries,
		retryWait: retryWait,
		log:       c.log.With("context", "osquery client connect"),
	}

	err = r.Run(ctx, func(_ context.Context) error {
		var err error
		cli, err = osquery.NewClient(c.socketPath, timeout)
		if err != nil {
			r.log.Warnf("failed to connect, reconnect might be attempted, err: %v", err)
			return err
		}
		return nil
	})
	return cli, err
}

func (c *Client) Close() {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.close()
}

func (c *Client) close() {
	if c.cli != nil {
		c.cli.Close()
		c.cli = nil
	}
}

// Query executes a given query, resolves the types
//
// In order to workaround the issue https://github.com/elastic/beats/issues/36622
// each query creates it's own RPC connection to osqueryd, allowing it to set a custom timeout per query.
// Current implementation of osqueryd RPC returns the error when the long running query times out, but this timeout is a transport timeout,
// that doesn't cancel the query execution itself.
// This also makes the client RPC unusable until the long running query finishes, returning errors for each subsequent query.
func (c *Client) Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]interface{}, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	err := c.cliLimiter.Acquire(ctx, limit)
	if err != nil {
		return nil, err
	}
	defer c.cliLimiter.Release(limit)

	// If query timeout is <= 0, then use client timeout (default is 1 minute)
	if timeout <= 0 {
		timeout = c.timeout
	}

	// If query timeout is greater that the maxTimeout, set it to the max timeout value
	if timeout > c.maxTimeout {
		timeout = c.maxTimeout
	}

	c.log.Debugf("osquery connect, query: %s, timeout: %v", sql, timeout)

	// Use a separate connection for queries in order to be able to recover from timed out queries
	cli, err := c.connectWithRetry(ctx, timeout)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	var res *genosquery.ExtensionResponse
	res, err = cli.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("osquery failed: %w", err)
	}

	if res.Status.Code != int32(0) {
		return nil, &ErrorQueryFailure{
			code:    res.Status.Code,
			message: res.Status.Message,
		}
	}

	return c.resolveResult(ctx, sql, res.Response)
}

// ResolveResult types for a give query
// The API is public to allow resolution of scheduled queries results captured by custom logger plugin
func (c *Client) ResolveResult(ctx context.Context, sql string, hits []map[string]string) ([]map[string]interface{}, error) {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.cli == nil {
		return nil, ErrClientClosed
	}

	err := c.cliLimiter.Acquire(ctx, limit)
	if err != nil {
		return nil, err
	}
	defer c.cliLimiter.Release(limit)

	return c.resolveResult(ctx, sql, hits)
}

func (c *Client) resolveResult(ctx context.Context, sql string, hits []map[string]string) ([]map[string]interface{}, error) {
	// Get column types
	colTypes, err := c.queryColumnTypes(ctx, sql)
	if err != nil {
		return nil, err
	}
	return resolveTypes(hits, colTypes), nil
}

func (c *Client) queryColumnTypes(ctx context.Context, sql string) (map[string]string, error) {
	var colTypes map[string]string

	if v, ok := c.cache.Get(sql); ok {
		colTypes, ok = v.(map[string]string)
		if ok {
			c.log.Debugf("using cached column types for query: %s", sql)
		} else {
			c.log.Error("failed get the column types from cache, incompatible type")
		}
	}

	if colTypes == nil {
		var (
			exres *genosquery.ExtensionResponse
			err   error
		)

		exres, err = c.cli.GetQueryColumnsContext(ctx, sql)

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
// If type conversion fails the value is preserved as string
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
					continue
				}
			case "UNSIGNED_BIGINT":
				var n uint64
				n, err = strconv.ParseUint(v, 10, 64)
				if err == nil {
					m[k] = n
					continue
				}
			case "DOUBLE":
				var n float64
				n, err = strconv.ParseFloat(v, 64)
				if err == nil {
					m[k] = n
					continue
				}
			}
		}
		// Keep the original string value if the value can not be converted
		m[k] = v
	}
	return m
}
