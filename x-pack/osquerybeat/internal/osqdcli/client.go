// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"
	"gotest.tools/gotestsum/log"

	"github.com/osquery/osquery-go"
	genosquery "github.com/osquery/osquery-go/gen/osquery"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

const (
	defaultTimeout        = 30 * time.Second
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
	socketPath     string
	timeout        time.Duration
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

	r := retry{
		maxRetry:  c.connectRetries,
		retryWait: retryWait,
		log:       c.log.With("context", "osquery client connect"),
	}

	return r.Run(ctx, func(ctx context.Context) error {
		cli, err := osquery.NewClient(c.socketPath, c.timeout)
		if err != nil {
			log.Errorf("failed to connect: %v", err)
			return err
		}
		c.cli = cli
		return nil
	})
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

func (c *Client) withReconnect(ctx context.Context, fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}

	var netErr *net.OpError

	// The current osquery go library github.com/osquery/osquery-go uses the older version of thrift library that
	// doesn't not wrap the original error, so we have to use this ugly check for the error message suffix here.
	// The latest version of thrift library is wrapping the error, so adding this check first here.
	if (errors.As(err, &netErr) && (netErr.Err == syscall.EPIPE || netErr.Err ==
		syscall.ECONNRESET)) ||
		strings.HasSuffix(err.Error(), " broken pipe") {

		c.log.Debugf("osquery error: %v, reconnect", err)

		// reconnect && retry
		err = c.reconnect(ctx)
		if err != nil {
			c.log.Errorf("failed to reconnect: %v", err)
			return err
		}
		return fn()
	}
	return nil
}

// Query executes a given query, resolves the types
func (c *Client) Query(ctx context.Context, sql string) ([]map[string]interface{}, error) {
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

	var res *genosquery.ExtensionResponse
	err = c.withReconnect(ctx, func() error {
		res, err = c.cli.Client.Query(ctx, sql)
		return err
	})
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

		err = c.withReconnect(ctx, func() error {
			exres, err = c.cli.Client.GetQueryColumns(ctx, sql)
			return err
		})

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
