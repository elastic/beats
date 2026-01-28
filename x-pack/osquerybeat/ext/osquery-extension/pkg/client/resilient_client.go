// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package client

import (
	"fmt"
	"time"

	"github.com/osquery/osquery-go"
	osquerygen "github.com/osquery/osquery-go/gen/osquery"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

// ResilientClient is an extension of the osquery ExtensionManagerClient
// that automatically attempts to reconnect if the connection is lost
type ResilientClient struct {
	socketPath string
	timeout    time.Duration
	log        *logger.Logger
	client     *osquery.ExtensionManagerClient
}

// NewResilientClient creates a new ResilientClient
func NewResilientClient(socketPath string, timeout time.Duration, log *logger.Logger) (*ResilientClient, error) {
	client, err := osquery.NewClient(socketPath, timeout)
	// if there is an error, we still create the ResilientClient
	// it will attempt to connect again on the first query
	if err != nil {
		log.Warningf("Could not create client to query osqueryd options: %s", err)
	}
	return &ResilientClient{
		socketPath: socketPath,
		timeout:    timeout,
		client:     client,
		log:        log,
	}, nil
}

// connect attempts to connect to the osqueryd socket if not already connected
func (rc *ResilientClient) connect() error {
	if rc.client != nil {
		return nil
	}

	client, err := osquery.NewClient(rc.socketPath, rc.timeout)
	if err != nil {
		return fmt.Errorf("could not create osquery client: %w", err)
	}

	rc.client = client
	return nil
}

// Close closes the underlying osquery client connection and ensures it is nil
func (rc *ResilientClient) Close() {
	if rc.client == nil {
		return
	}
	rc.client.Close()
	rc.client = nil
}

// Options retrieves the osqueryd options via the osquery ExtensionManagerClient
func (rc *ResilientClient) Options() (osquerygen.InternalOptionList, error) {
	if err := rc.connect(); err != nil {
		return nil, err
	}
	options, err := rc.client.Options()
	if err != nil {
		rc.Close()
		return nil, err
	}
	return options, nil
}

// Query executes the given SQL query against osqueryd via the osquery ExtensionManagerClient
func (rc *ResilientClient) Query(sql string) (*osquerygen.ExtensionResponse, error) {
	// connect if needed
	if err := rc.connect(); err != nil {
		return nil, err
	}

	// execute the query
	response, err := rc.client.Query(sql)
	if err != nil {
		rc.log.Warningf("Failed to execute osqueryd query: %s", err)
		rc.Close()
		return nil, err
	}
	return response, nil
}
