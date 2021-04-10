// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqueryd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/kolide/osquery-go"
)

const (
	DefaultTimeout = 30 * time.Second

	retryWait  = 200 * time.Millisecond
	retryTimes = 10
	logTag     = "osqueryd_cli"
)

type Client struct {
	cli *osquery.ExtensionManagerClient
}

func NewClient(ctx context.Context, socketPath string, to time.Duration) (*Client, error) {
	cli, err := newClientWithRetries(ctx, socketPath, to)
	if err != nil {
		return nil, err
	}
	return &Client{
		cli: cli,
	}, nil
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

func (c *Client) Query(ctx context.Context, sql string) ([]map[string]string, error) {
	res, err := c.cli.Client.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("osquery failed: %w", err)
	}
	if res.Status.Code != int32(0) {
		return nil, errors.New(res.Status.Message)
	}
	return res.Response, nil
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
