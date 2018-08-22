// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
)

const defaultTimeout = 10 * time.Second

// Client to Central Management API
type Client struct {
	client *kibana.Client
}

// ConfigFromURL generates a full kibana client config from an URL
func ConfigFromURL(kibanaURL string) (*kibana.ClientConfig, error) {
	data, err := url.Parse(kibanaURL)
	if err != nil {
		return nil, err
	}

	var username, password string
	if data.User != nil {
		username = data.User.Username()
		password, _ = data.User.Password()
	}

	return &kibana.ClientConfig{
		Protocol: data.Scheme,
		Host:     data.Host,
		Path:     data.Path,
		Username: username,
		Password: password,
		Timeout:  defaultTimeout,
	}, nil
}

// NewClient creates and returns a kibana client
func NewClient(cfg *kibana.ClientConfig) (*Client, error) {
	client, err := kibana.NewClientWithConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: client,
	}, nil
}

// do a request to the API and unmarshall the message, error if anything fails
func (c *Client) request(method, extraPath string,
	params common.MapStr, headers http.Header, message interface{}) (int, error) {

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return 400, err
	}

	statusCode, result, err := c.client.Request(method, extraPath, nil, headers, bytes.NewBuffer(paramsJSON))
	if err != nil {
		return statusCode, err
	}

	if statusCode >= 300 {
		err = extractError(result)
	} else {
		if err = json.Unmarshal(result, message); err != nil {
			return statusCode, errors.Wrap(err, "error unmarshaling response")
		}
	}

	return statusCode, err
}

func extractError(result []byte) error {
	var kibanaResult struct {
		Message string
	}
	if err := json.Unmarshal(result, &kibanaResult); err != nil {
		return errors.Wrap(err, "parsing kibana response")
	}
	return errors.New(kibanaResult.Message)
}
