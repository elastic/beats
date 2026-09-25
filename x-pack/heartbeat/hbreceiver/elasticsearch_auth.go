// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionauth"

	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
)

const elasticsearchRequestTimeout = 10 * time.Second

type elasticsearchAuthExtension interface {
	extensionauth.HTTPClient
	Endpoints() []string
}

type esClient struct {
	endpoints []*url.URL
	client    *http.Client
	next      atomic.Uint64
}

var _ monitorstate.ElasticsearchRequester = (*esClient)(nil)

func elasticsearchAuthStartHook(reference string, heartbeat *beater.Heartbeat, setRequester func(*esClient)) func(component.Host) error {
	return func(host component.Host) error {
		if reference == "" {
			return nil
		}

		var extensionID component.ID
		if err := extensionID.UnmarshalText([]byte(reference)); err != nil {
			return fmt.Errorf("invalid elasticsearch_auth component ID %q: %w", reference, err)
		}

		extension, ok := host.GetExtensions()[extensionID]
		if !ok {
			return fmt.Errorf("elasticsearch_auth extension %q not found", extensionID.String())
		}
		auth, ok := extension.(elasticsearchAuthExtension)
		if !ok {
			return fmt.Errorf(
				"elasticsearch_auth extension %q has type %T, which does not implement HTTP authentication and endpoint discovery",
				extensionID.String(),
				extension,
			)
		}

		if heartbeat == nil {
			return fmt.Errorf("heartbeat instance was not captured for elasticsearch_auth extension %q", extensionID.String())
		}

		requester, err := newESClient(auth)
		if err != nil {
			return fmt.Errorf("creating Elasticsearch requester from extension %q: %w", extensionID.String(), err)
		}
		heartbeat.WithElasticsearchStateLoader(requester)
		setRequester(requester)
		return nil
	}
}

func newESClient(auth elasticsearchAuthExtension) (*esClient, error) {
	endpoints := auth.Endpoints()
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("extension has no endpoints")
	}

	parsedEndpoints := make([]*url.URL, 0, len(endpoints))
	for _, endpoint := range endpoints {
		parsedEndpoint, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("parsing endpoint %q: %w", endpoint, err)
		}
		parsedEndpoints = append(parsedEndpoints, parsedEndpoint)
	}

	roundTripper, err := auth.RoundTripper(http.DefaultTransport)
	if err != nil {
		return nil, fmt.Errorf("creating authenticated transport: %w", err)
	}
	return &esClient{
		endpoints: parsedEndpoints,
		// elasticsearchauth intentionally does not own request deadlines; Heartbeat
		// keeps the 10-second deadline used by its prior Elasticsearch requester.
		client: &http.Client{
			Transport: roundTripper,
			Timeout:   elasticsearchRequestTimeout,
		},
	}, nil
}

// Request implements monitorstate.ElasticsearchRequester with the authenticated
// HTTP transport and endpoints exposed by elasticsearchauth.
func (e *esClient) Request(method, path, pipeline string, params map[string]string, body any) (int, []byte, error) {
	var encodedBody []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("encoding Elasticsearch request body: %w", err)
		}
		encodedBody = encoded
	}

	//nolint:gosec // It's the number of Elasticsearch hosts, it won't overflow
	start := int(e.next.Add(1)-1) % len(e.endpoints)
	var requestErr error
	for offset := range len(e.endpoints) {
		endpoint := requestURL(e.endpoints[(start+offset)%len(e.endpoints)], path, pipeline, params)
		//nolint:noctx // The interface we're implementing does not accept a context and the HTTP client has a timeout set
		request, err := http.NewRequest(method, endpoint.String(), bytes.NewReader(encodedBody))
		if err != nil {
			return 0, nil, fmt.Errorf("creating Elasticsearch request: %w", err)
		}
		if body != nil {
			request.Header.Set("Content-Type", "application/json")
		}

		response, err := e.client.Do(request)
		if err != nil {
			requestErr = err
			continue
		}
		responseBody, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return response.StatusCode, nil, fmt.Errorf("reading Elasticsearch response: %w", readErr)
		}
		if closeErr != nil {
			return response.StatusCode, nil, fmt.Errorf("closing Elasticsearch response: %w", closeErr)
		}
		return response.StatusCode, responseBody, nil
	}
	return 0, nil, fmt.Errorf("requesting Elasticsearch endpoints: %w", requestErr)
}

func requestURL(endpoint *url.URL, path, pipeline string, params map[string]string) *url.URL {
	requestURL := *endpoint
	requestPath, err := url.Parse(path)
	if err != nil {
		requestPath = &url.URL{Path: path}
	}

	requestURL.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(requestPath.Path, "/")

	query := requestURL.Query()
	maps.Copy(query, requestPath.Query())

	if pipeline != "" {
		query.Set("pipeline", pipeline)
	}
	for key, value := range params {
		query.Set(key, value)
	}
	requestURL.RawQuery = query.Encode()
	return &requestURL
}

func (e *esClient) CloseIdleConnections() {
	if closer, ok := e.client.Transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
