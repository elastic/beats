// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"

	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/elastic-agent-libs/logp"
)

const elasticsearchRequestTimeout = 10 * time.Second

type beatsAuthExtension interface {
	extensionauth.HTTPClient
	Endpoints() []string
}

var _ monitorstate.ElasticsearchRequester = (*eslegclient.Connection)(nil)

func beatsAuthStartHook(
	ctx context.Context,
	reference string,
	heartbeat *beater.Heartbeat,
	userAgent string,
	logger *logp.Logger,
	setRequester func(*eslegclient.Connection),
) func(component.Host) error {

	return func(host component.Host) error {
		if reference == "" {
			return nil
		}

		var extensionID component.ID
		if err := extensionID.UnmarshalText([]byte(reference)); err != nil {
			return fmt.Errorf("invalid monitor_state.elasticsearch component ID %q: %w", reference, err)
		}

		extension, ok := host.GetExtensions()[extensionID]
		if !ok {
			return fmt.Errorf("monitor_state.elasticsearch extension %q not found", extensionID.String())
		}
		auth, ok := extension.(beatsAuthExtension)
		if !ok {
			return fmt.Errorf(
				"monitor_state.elasticsearch extension %q has type %T, which does not implement HTTP authentication and endpoint discovery",
				extensionID.String(),
				extension,
			)
		}

		if heartbeat == nil {
			return fmt.Errorf("heartbeat instance was not captured for monitor_state.elasticsearch extension %q", extensionID.String())
		}

		requester, err := newESClient(ctx, auth, userAgent, logger)
		if err != nil {
			return fmt.Errorf("creating Elasticsearch requester from extension %q: %w", extensionID.String(), err)
		}
		heartbeat.WithElasticsearchStateLoader(requester)
		setRequester(requester)
		return nil
	}
}

func newESClient(
	ctx context.Context,
	auth beatsAuthExtension,
	userAgent string,
	logger *logp.Logger,
) (*eslegclient.Connection, error) {
	endpoints := auth.Endpoints()

	roundTripper, err := auth.RoundTripper(nil)
	if err != nil {
		return nil, fmt.Errorf("creating authenticated transport: %w", err)
	}

	connectionErrors := make([]string, 0, len(endpoints))
	var fallbackClient *eslegclient.Connection
	for _, endpoint := range endpoints {
		client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
			URL:       endpoint,
			Beatname:  "Heartbeat",
			UserAgent: userAgent,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("create Elasticsearch client for endpoint %q: %w", endpoint, err)
		}
		client.Headers["User-Agent"] = client.UserAgent

		// beatsauth owns authentication and transport configuration.
		// Heartbeat owns the request deadline that applied to its prior client.
		client.HTTP = &http.Client{
			Transport: roundTripper,
			Timeout:   elasticsearchRequestTimeout,
		}
		if err := client.Connect(ctx); err != nil {
			connectionErrors = append(connectionErrors, err.Error())
			if fallbackClient == nil {
				fallbackClient = client
			}
			continue
		}

		return client, nil
	}

	if fallbackClient != nil {
		logger.Warnf(
			"couldn't connect to any Elasticsearch authenticator endpoint; "+
				"Heartbeat will continue without previous monitor state until "+
				"connectivity returns: %s",
			strings.Join(connectionErrors, "; "),
		)
		return fallbackClient, nil
	}

	return nil, fmt.Errorf(
		"couldn't connect to any Elasticsearch authenticator endpoint: %s",
		strings.Join(connectionErrors, "; "),
	)
}
