// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type FakeObject struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func TestDeserializeDataFails(t *testing.T) {
	var invalidJson = []string{`{"id":true}`, `{"malformed"}`}

	for _, json := range invalidJson {
		_, err := DeserializeData[FakeObject]([]byte(json))

		require.ErrorContains(t, err, "failed to deserialize data")
	}
}

func TestDeserializeDataSucceeds(t *testing.T) {
	var validJson = []string{`{}`, `{"id": "123"}`, `{"id":"456","name":"the name","other":"field"}`}

	for _, json := range validJson {
		obj, err := DeserializeData[FakeObject]([]byte(json))

		require.NoError(t, err)
		require.NotNil(t, obj)
	}

	obj, err := DeserializeData[FakeObject]([]byte(validJson[2]))

	require.NoError(t, err)
	require.Equal(t, "456", obj.Id)
	require.Equal(t, "the name", obj.Name)
}

func createHttpResponse(statusCode int, status string, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func getHttpResponseError(t *testing.T, resp *http.Response, requestError error) *HTTPResponse {
	_, err := HandleHTTPResponse[FakeObject](resp, requestError)

	e, ok := err.(*HTTPResponse)

	require.True(t, ok)

	return e
}

func TestDeserializeErrorResponseFailedToSend(t *testing.T) {
	error := fmt.Errorf("test error")

	err := getHttpResponseError(t, nil, error)

	require.Error(t, err)
	require.Equal(t, 0, err.StatusCode)
	require.Equal(t, "failed to send request", err.Status)
	require.Equal(t, "", err.Body)
	require.Errorf(t, err.Err, "failed to send request: %w", error)
}

func TestDeserializeErrorElasticsearchResponse(t *testing.T) {
	status := "error_type"
	message := "error message"

	var responses = []string{
		`{"error":{"type":"` + status + `","reason":"` + message + `"},"status":1}`,
		`{"error":{"type":"` + status + `","reason":"` + message + `","root_cause":[]},"status":2}`,
		`{"error":{"type":"` + status + `","reason":"` + message + `","root_cause":[{"type":"ignored","reason":"ignored"}]},"status":3}`,
	}

	// Elasticsearch error responses
	for i, json := range responses {
		response := createHttpResponse(400+i, "ignored for ES", json)

		err := getHttpResponseError(t, response, nil)

		require.Error(t, err)
		require.Equal(t, 400+i, err.StatusCode)
		require.Equal(t, status, err.Status)
		require.Equal(t, message, err.Body)
		require.Errorf(t, err.Err, "error from Elasticsearch [%s]: %s", status, message)
	}
}

func TestDeserializeErrorCloudConnectResponse(t *testing.T) {
	status := "error_code"
	message := "error message"

	var responses = []string{
		`{"errors":[{"code":"` + status + `","message":"` + message + `"}]}`,
		`{"errors":[{"code":"` + status + `","message":"` + message + `"},{"code":"ignored","message":"ignored"}]}`,
	}

	// Cloud Connect error responses
	for i, json := range responses {
		response := createHttpResponse(400+i, "ignored for CC", json)

		err := getHttpResponseError(t, response, nil)

		require.Error(t, err)
		require.Equal(t, 400+i, err.StatusCode)
		require.Equal(t, status, err.Status)
		require.Equal(t, message, err.Body)
		require.Errorf(t, err.Err, "error from Cloud Connect API [%s]: %s", status, message)
	}
}

func TestDeserializeErrorUnknownResponse(t *testing.T) {
	var responses = []string{
		`{"some_field":"some_value"}`,
		`{"malformed"}`,
		`{`, // malformed JSON
		`Not a JSON response`,
		``,
	}

	// Unknown error responses
	for _, json := range responses {
		response := createHttpResponse(400, "BAD_REQUEST 400", json)

		err := getHttpResponseError(t, response, nil)

		require.Error(t, err)
		require.Equal(t, 400, err.StatusCode)
		require.Equal(t, "BAD_REQUEST 400", err.Status)
		require.Equal(t, json, err.Body)
		require.Errorf(t, err.Err, "failed to fetch data: HTTP status %d with body %s", 400, json)
	}
}

func TestHandleHTTPResponseDeserializationFails(t *testing.T) {
	var responses = []string{
		`{"id": true}`,
		`{"malformed"}`,
		`{`,
	}

	for _, json := range responses {
		response := createHttpResponse(200, "OK 200", json)

		err := getHttpResponseError(t, response, nil)

		require.Equal(t, 200, err.StatusCode)
		require.Equal(t, "failed to deserialize data", err.Status)
		require.ErrorContains(t, err, "failed to deserialize data")
	}
}

func TestHandleHTTPResponseSucceeds(t *testing.T) {
	var responses = []string{
		`{"id":"123","name":"test name"}`,
		`{"id":"456"}`,
		`{}`,
	}

	for _, json := range responses {
		response := createHttpResponse(200, "OK 200", json)

		obj, err := HandleHTTPResponse[FakeObject](response, nil)

		require.NoError(t, err)
		require.NotNil(t, obj)
	}
}
