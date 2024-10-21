// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ess

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	config *Config
	client *http.Client
}

func NewClient(config Config) *Client {
	cfg := defaultConfig()
	cfg.Merge(config)

	c := new(Client)
	c.client = http.DefaultClient
	c.config = cfg

	return c
}

func (c *Client) doGet(ctx context.Context, relativeUrl string) (*http.Response, error) {
	u, err := url.JoinPath(c.config.BaseUrl, relativeUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to create API URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create GET request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", c.config.ApiKey))

	return c.client.Do(req)
}

func (c *Client) doPost(ctx context.Context, relativeUrl, contentType string, body io.Reader) (*http.Response, error) {
	u, err := url.JoinPath(c.config.BaseUrl, relativeUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to create API URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, fmt.Errorf("unable to create POST request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", c.config.ApiKey))
	req.Header.Set("Content-Type", contentType)

	return c.client.Do(req)
}

func (c *Client) BaseURL() string {
	return c.config.BaseUrl
}
