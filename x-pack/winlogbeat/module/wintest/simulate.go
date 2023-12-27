// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package wintest

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

var KeepRunning = flag.Bool("keep-running", false, "don't tear down simulate docker (will print command to manually stop instance)")

// TestCase is a file input and Elasticsearch response set returned by SimulatePipeline.
type TestCase struct {
	Path      string
	Collected []json.RawMessage
	Processed []json.RawMessage
	Err       error
}

// SimulatePipeline runs the Elasticsearch simulate pipeline on the provided host using
// user and pass as authentication. The pipeline used must already exist in the elasticsearch
// instance. The paths is the set of JSON documents to send to simulate.
//
// The returned test cases will contain the name of the input file, the input data,
// the resulting processed documents and any Elasticsearch error messages. If error
// is non-nil, the returned test cases are not valid.
func SimulatePipeline(host, user, pass, pipeline string, paths []string) ([]TestCase, error) {
	if host == "" {
		return nil, errors.New("missing required host name")
	}

	cases, err := readRawTestData(paths)
	if err != nil {
		return nil, err
	}

	config := elasticsearch.Config{
		Addresses: []string{host},
		Username:  user,
		Password:  pass,
	}
	client, err := elasticsearch.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to make client: %w", err)
	}

	for i, k := range cases {
		cases[i].Processed, cases[i].Err = simulatePipeline(client.API, pipeline, k.Collected)
		for j := range k.Collected {
			cases[i].Collected[j], err = marshalNormalizedJSON(cases[i].Collected[j])
			if err != nil {
				return nil, err
			}
		}
		for j := range cases[i].Processed {
			cases[i].Processed[j], err = marshalNormalizedJSON(cases[i].Processed[j])
			if err != nil {
				return nil, err
			}
		}
	}
	return cases, nil
}

// readRawTestData loads the unprocessed data held in the provided paths.
func readRawTestData(paths []string) ([]TestCase, error) {
	var cases []TestCase
	for _, path := range paths {
		events, err := readEvents(path)
		if err != nil {
			return nil, err
		}
		cases = append(cases, TestCase{
			Path:      path,
			Collected: events,
		})
	}
	return cases, nil
}

func readEvents(path string) ([]json.RawMessage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var events []json.RawMessage
	err = json.Unmarshal(b, &events)
	return events, err
}

// simulatePipeline runs a single simulate query on the specified pipeline
// with the provided documents.
func simulatePipeline(api *esapi.API, pipeline string, docs []json.RawMessage) ([]json.RawMessage, error) {
	var request simulatePipelineRequest
	for _, event := range docs {
		request.Docs = append(request.Docs, pipelineDocument{
			Source: event,
		})
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling simulate request failed: %w", err)
	}

	resp, err := api.Ingest.Simulate(bytes.NewReader(requestBody), func(request *esapi.IngestSimulateRequest) {
		request.PipelineID = pipeline
	})
	if err != nil {
		return nil, fmt.Errorf("failed to simulate %q pipeline: %w", pipeline, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read simulate response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status for simulate: %s (%d): %w", resp.Status(), resp.StatusCode, newError(body))
	}

	var response simulatePipelineResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal simulate response: %w", err)
	}
	var events []json.RawMessage
	for _, doc := range response.Docs {
		events = append(events, doc.Doc.Source)
	}
	return events, nil
}

type simulatePipelineRequest struct {
	Docs []pipelineDocument `json:"docs"`
}

type simulatePipelineResponse struct {
	Docs []pipelineIngestedDocument `json:"docs"`
}

type pipelineIngestedDocument struct {
	Doc pipelineDocument `json:"doc"`
}

type pipelineDocument struct {
	Source json.RawMessage `json:"_source"`
}

// newError returns a new error constructed from the given response body.
// This assumes the body contains a JSON encoded error. If the body cannot
// be parsed then an error is returned that contains the raw body.
func newError(body []byte) error {
	var msg struct {
		Error struct {
			RootCause []struct {
				Type          string   `json:"type"`
				Reason        string   `json:"reason"`
				ProcessorType string   `json:"processor_type,omitempty"`
				ScriptStack   []string `json:"script_stack,omitempty"`
				Script        string   `json:"script,omitempty"`
				Lang          string   `json:"lang,omitempty"`
				Position      struct {
					Offset int `json:"offset"`
					Start  int `json:"start"`
					End    int `json:"end"`
				} `json:"position,omitempty"`
				Suppressed []struct {
					Type          string `json:"type"`
					Reason        string `json:"reason"`
					ProcessorType string `json:"processor_type"`
				} `json:"suppressed,omitempty"`
			} `json:"root_cause,omitempty"`
			Type          string   `json:"type"`
			Reason        string   `json:"reason"`
			ProcessorType string   `json:"processor_type,omitempty"`
			ScriptStack   []string `json:"script_stack,omitempty"`
			Script        string   `json:"script,omitempty"`
			Lang          string   `json:"lang,omitempty"`
			Position      struct {
				Offset int `json:"offset"`
				Start  int `json:"start"`
				End    int `json:"end"`
			} `json:"position,omitempty"`
			CausedBy struct {
				Type     string `json:"type"`
				Reason   string `json:"reason"`
				CausedBy struct {
					Type   string      `json:"type"`
					Reason interface{} `json:"reason"`
				} `json:"caused_by,omitempty"`
			} `json:"caused_by,omitempty"`
			Suppressed []struct {
				Type          string `json:"type"`
				Reason        string `json:"reason"`
				ProcessorType string `json:"processor_type"`
			} `json:"suppressed,omitempty"`
		} `json:"error"`
		Status int `json:"status"`
	}

	err := json.Unmarshal(body, &msg)
	if err != nil {
		// Fall back to including to raw body if it cannot be parsed.
		return fmt.Errorf("elasticsearch error: %s", body)
	}
	if len(msg.Error.RootCause) > 0 {
		cause, _ := json.MarshalIndent(msg.Error.RootCause, "", "  ")
		return fmt.Errorf("elasticsearch error (type=%s): %s\nRoot cause:\n%s", msg.Error.Type, msg.Error.Reason, cause)
	}
	return fmt.Errorf("elasticsearch error (type=%s): %s", msg.Error.Type, msg.Error.Reason)
}

// marshalNormalizedJSON marshals test results ensuring that field
// order remains consistent independent of field order returned by
// ES to minimize diff noise during changes.
func marshalNormalizedJSON(v interface{}) ([]byte, error) {
	msg, err := json.Marshal(v)
	if err != nil {
		return msg, err
	}
	var obj interface{}
	err = jsonUnmarshalUsingNumber(msg, &obj)
	if err != nil {
		return msg, err
	}
	return json.MarshalIndent(obj, "", "    ")
}

// jsonUnmarshalUsingNumber is a drop-in replacement for json.Unmarshal that
// does not default to unmarshaling numeric values to float64 in order to
// prevent low bit truncation of values greater than 1<<53.
// See https://golang.org/cl/6202068 for details.
func jsonUnmarshalUsingNumber(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := dec.Decode(v)
	if err != nil {
		if err == io.EOF { //nolint:errorlint // Bad linter! // io.EOF is never wrapped.
			return errors.New("unexpected end of JSON input")
		}
		return err
	}
	// Make sure there is no more data after the message
	// to approximate json.Unmarshal's behaviour.
	if dec.More() {
		return fmt.Errorf("more data after top-level value")
	}
	return nil
}

// ErrorMessage returns any Elasticsearch error.message in the provided
// JSON data.
func ErrorMessage(msg json.RawMessage) error {
	var event struct {
		Error struct {
			Message interface{}
		}
	}
	err := json.Unmarshal(msg, &event)
	if err != nil {
		return fmt.Errorf("can't unmarshal event to check pipeline error: %#q: %w", msg, err)
	}

	switch m := event.Error.Message.(type) {
	case nil:
		return nil
	case string, []string:
		return fmt.Errorf("unexpected pipeline error: %s", m)
	default:
		return fmt.Errorf("unexpected pipeline error (unexpected error.message type %T): %[1]v", m)
	}
}
